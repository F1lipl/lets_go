package respository

import (
	"contentserver/internal/event"
	"context"
	"database/sql"
	"errors"
	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"os"
	"testing"
	"time"
)

func TestTaskExecutionMySQL(t *testing.T) {
	dsn := os.Getenv("CONTENT_MODEL_TEST_DSN")
	if dsn == "" {
		t.Skip("set CONTENT_MODEL_TEST_DSN")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	conn := sqlx.NewSqlConnFromDB(db)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	id, token := uuid.NewString(), uuid.NewString()
	defer func() {
		_, _ = conn.ExecCtx(context.Background(), "DELETE FROM event_delivery WHERE event_id=?", id)
		_, _ = conn.ExecCtx(context.Background(), "DELETE FROM outbox_event WHERE event_id=?", id)
	}()
	if _, err = conn.ExecCtx(ctx, "INSERT INTO outbox_event(event_id,aggregate_type,aggregate_id,event_type,payload_json,occurred_at) VALUES (?,'Test',?,'Test','{\"schemaVersion\":1}',NOW(3))", id, id); err != nil {
		t.Fatal(err)
	}
	if _, err = conn.ExecCtx(ctx, "INSERT INTO event_delivery(event_id,consumer_name,status,attempt_count,claim_token,locked_until) VALUES (?,'test',2,1,?,TIMESTAMPADD(SECOND,60,NOW(3)))", id, token); err != nil {
		t.Fatal(err)
	}
	task := event.ClaimedTask{Context: event.TaskContext{EventId: id, ConsumerName: "test"}, ClaimToken: token}
	repo := NewTaskRepository(conn)
	sentinel := errors.New("business failure")
	err = repo.Execute(ctx, task, func(ctx context.Context, tx sqlx.Session, input *event.TaskContext) error {
		if input.EventType != "Test" || input.SchemaVersion != 1 {
			return errors.New("event not loaded")
		}
		if _, err := tx.ExecCtx(ctx, "UPDATE outbox_event SET aggregate_type='Changed' WHERE event_id=?", id); err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("callback error: %v", err)
	}
	var kind string
	if err := conn.QueryRowCtx(ctx, &kind, "SELECT aggregate_type FROM outbox_event WHERE event_id=?", id); err != nil || kind != "Test" {
		t.Fatalf("rollback: %s %v", kind, err)
	}
	if err := repo.ScheduleRetry(ctx, task, sentinel, time.Second, false); err != nil {
		t.Fatal(err)
	}
	var status int
	if err := conn.QueryRowCtx(ctx, &status, "SELECT status FROM event_delivery WHERE event_id=?", id); err != nil || status != 1 {
		t.Fatalf("retry: %d %v", status, err)
	}
	oldToken := token
	token = uuid.NewString()
	if _, err := conn.ExecCtx(ctx, "UPDATE event_delivery SET status=2,claim_token=?,locked_until=TIMESTAMPADD(SECOND,60,NOW(3)),attempt_count=2 WHERE event_id=?", token, id); err != nil {
		t.Fatal(err)
	}
	if err := repo.Execute(ctx, task, func(context.Context, sqlx.Session, *event.TaskContext) error {
		t.Error("stale claim invoked handler")
		return nil
	}); !errors.Is(err, event.ErrClaimLost) {
		t.Fatalf("stale claim: %v", err)
	}
	task.ClaimToken = token
	if err := repo.Execute(ctx, task, func(ctx context.Context, tx sqlx.Session, _ *event.TaskContext) error {
		_, err := tx.ExecCtx(ctx, "UPDATE outbox_event SET aggregate_type='Changed' WHERE event_id=?", id)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := conn.QueryRowCtx(ctx, &status, "SELECT status FROM event_delivery WHERE event_id=?", id); err != nil || status != 3 {
		t.Fatalf("completion: %d %v", status, err)
	}
	task.ClaimToken = oldToken
	if err := repo.ScheduleRetry(ctx, task, sentinel, time.Second, false); !errors.Is(err, event.ErrClaimLost) {
		t.Fatalf("old retry modified completed task: %v", err)
	}
}

func TestTaskClaimMySQL(t *testing.T) {
	dsn := os.Getenv("CONTENT_MODEL_TEST_DSN")
	if dsn == "" {
		t.Skip("set CONTENT_MODEL_TEST_DSN")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	conn := sqlx.NewSqlConnFromDB(db)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	ids := []string{uuid.NewString(), uuid.NewString(), uuid.NewString()}
	defer func() {
		for _, id := range ids {
			_, _ = conn.ExecCtx(context.Background(), "DELETE FROM event_delivery WHERE event_id=?", id)
			_, _ = conn.ExecCtx(context.Background(), "DELETE FROM outbox_event WHERE event_id=?", id)
		}
	}()
	for i, id := range ids {
		if _, err := conn.ExecCtx(ctx, "INSERT INTO outbox_event(event_id,aggregate_type,aggregate_id,event_type,payload_json,occurred_at) VALUES (?,'Test',?,'Test','{\"schemaVersion\":1}',NOW(3))", id, id); err != nil {
			t.Fatal(err)
		}
		nextAttempt := "NOW(3)"
		if i == 2 {
			nextAttempt = "TIMESTAMPADD(HOUR,1,NOW(3))"
		}
		if _, err := conn.ExecCtx(ctx, "INSERT INTO event_delivery(event_id,consumer_name,next_attempt_at) VALUES (?,'test',"+nextAttempt+")", id); err != nil {
			t.Fatal(err)
		}
	}

	repo := NewTaskRepository(conn)
	first, err := repo.Claim(ctx, 1, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	second, err := repo.Claim(ctx, 2, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("claimed batches = %d and %d, want 1 and 1", len(first), len(second))
	}
	if first[0].Context.EventId == second[0].Context.EventId {
		t.Fatal("same delivery was claimed twice")
	}
	if first[0].ClaimToken == "" || second[0].ClaimToken == "" {
		t.Fatal("claim token was not created")
	}
	if !first[0].LockedUntil.After(time.Now()) || !second[0].LockedUntil.After(time.Now()) {
		t.Fatal("claim lease was not persisted")
	}

	var processing int
	if err := conn.QueryRowCtx(ctx, &processing, "SELECT COUNT(*) FROM event_delivery WHERE event_id IN (?,?,?) AND status=2 AND attempt_count=1 AND claim_token IS NOT NULL AND locked_until IS NOT NULL", ids[0], ids[1], ids[2]); err != nil {
		t.Fatal(err)
	}
	if processing != 2 {
		t.Fatalf("processing deliveries = %d, want 2", processing)
	}
	remaining, err := repo.Claim(ctx, 3, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 0 {
		t.Fatalf("future delivery was claimed: %d", len(remaining))
	}
}

func TestRecoverExpiredTasksMySQL(t *testing.T) {
	dsn := os.Getenv("CONTENT_MODEL_TEST_DSN")
	if dsn == "" {
		t.Skip("set CONTENT_MODEL_TEST_DSN")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	conn := sqlx.NewSqlConnFromDB(db)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	ids := []string{uuid.NewString(), uuid.NewString(), uuid.NewString()}
	defer func() {
		for _, id := range ids {
			_, _ = conn.ExecCtx(context.Background(), "DELETE FROM event_delivery WHERE event_id=?", id)
			_, _ = conn.ExecCtx(context.Background(), "DELETE FROM outbox_event WHERE event_id=?", id)
		}
	}()
	for _, id := range ids {
		if _, err := conn.ExecCtx(ctx, "INSERT INTO outbox_event(event_id,aggregate_type,aggregate_id,event_type,payload_json,occurred_at) VALUES (?,'Test',?,'Test','{\"schemaVersion\":1}',NOW(3))", id, id); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := conn.ExecCtx(ctx, `
INSERT INTO event_delivery(event_id,consumer_name,status,attempt_count,max_attempts,claim_token,locked_until)
VALUES
  (?,'recover',2,1,3,?,TIMESTAMPADD(SECOND,-1,NOW(3))),
  (?,'fail',2,3,3,?,TIMESTAMPADD(SECOND,-1,NOW(3))),
  (?,'active',2,1,3,?,TIMESTAMPADD(MINUTE,1,NOW(3)))`,
		ids[0], uuid.NewString(), ids[1], uuid.NewString(), ids[2], uuid.NewString()); err != nil {
		t.Fatal(err)
	}

	result, err := NewTaskRepository(conn).RecoverExpired(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if result != (event.RecoveryResult{Scanned: 2, Recovered: 1, Failed: 1}) {
		t.Fatalf("unexpected recovery result: %+v", result)
	}

	type state struct {
		Status       uint64         `db:"status"`
		AttemptCount uint64         `db:"attempt_count"`
		ClaimToken   sql.NullString `db:"claim_token"`
		LockedUntil  sql.NullTime   `db:"locked_until"`
		CompletedAt  sql.NullTime   `db:"completed_at"`
	}
	var recovered, failed, active state
	if err := conn.QueryRowCtx(ctx, &recovered, "SELECT status,attempt_count,claim_token,locked_until,completed_at FROM event_delivery WHERE event_id=?", ids[0]); err != nil {
		t.Fatal(err)
	}
	if recovered.Status != 1 || recovered.AttemptCount != 1 || recovered.ClaimToken.Valid || recovered.LockedUntil.Valid || recovered.CompletedAt.Valid {
		t.Fatalf("recovered task state: %+v", recovered)
	}
	if err := conn.QueryRowCtx(ctx, &failed, "SELECT status,attempt_count,claim_token,locked_until,completed_at FROM event_delivery WHERE event_id=?", ids[1]); err != nil {
		t.Fatal(err)
	}
	if failed.Status != 5 || failed.AttemptCount != 3 || failed.ClaimToken.Valid || failed.LockedUntil.Valid || !failed.CompletedAt.Valid {
		t.Fatalf("failed task state: %+v", failed)
	}
	if err := conn.QueryRowCtx(ctx, &active, "SELECT status,attempt_count,claim_token,locked_until,completed_at FROM event_delivery WHERE event_id=?", ids[2]); err != nil {
		t.Fatal(err)
	}
	if active.Status != 2 || active.AttemptCount != 1 || !active.ClaimToken.Valid || !active.LockedUntil.Valid || active.CompletedAt.Valid {
		t.Fatalf("active task state: %+v", active)
	}
}
