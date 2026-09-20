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
