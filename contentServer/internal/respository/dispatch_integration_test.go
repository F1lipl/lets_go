package respository

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"contentserver/internal/event"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type duplicateConsumerResolver struct{}

func (duplicateConsumerResolver) Consumers(string) ([]string, bool) {
	return []string{"duplicate", "duplicate"}, true
}

func TestDispatchFanOutMySQL(t *testing.T) {
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

	knownID := uuid.NewString()
	unknownID := uuid.NewString()
	ids := []string{knownID, unknownID}
	defer func() {
		for _, id := range ids {
			_, _ = conn.ExecCtx(context.Background(), "DELETE FROM event_delivery WHERE event_id=?", id)
			_, _ = conn.ExecCtx(context.Background(), "DELETE FROM outbox_event WHERE event_id=?", id)
		}
	}()
	if _, err := conn.ExecCtx(ctx, "INSERT INTO outbox_event(event_id,aggregate_type,aggregate_id,event_type,payload_json,occurred_at) VALUES (?,'Post',?,'PostPublished','{\"schemaVersion\":1}',NOW(3)),(?,'Post',?,'UnknownEvent','{\"schemaVersion\":1}',NOW(3))", knownID, knownID, unknownID, unknownID); err != nil {
		t.Fatal(err)
	}

	registry, err := event.NewConsumerRegistry(map[string][]string{
		"PostPublished": {"post_card", "post_search"},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := NewDispatchRepository(conn).FanOut(ctx, 10, registry)
	if err != nil {
		t.Fatal(err)
	}
	if result.ScannedEvents != 2 || result.DispatchedEvents != 1 || result.FailedEvents != 1 || result.CreatedTasks != 2 {
		t.Fatalf("unexpected dispatch result: %+v", result)
	}

	var taskCount int
	if err := conn.QueryRowCtx(ctx, &taskCount, "SELECT COUNT(*) FROM event_delivery WHERE event_id=? AND status=1", knownID); err != nil {
		t.Fatal(err)
	}
	if taskCount != 2 {
		t.Fatalf("created tasks = %d, want 2", taskCount)
	}
	var known struct {
		Status       uint64       `db:"status"`
		AttemptCount uint64       `db:"attempt_count"`
		PublishedAt  sql.NullTime `db:"published_at"`
	}
	if err := conn.QueryRowCtx(ctx, &known, "SELECT status,attempt_count,published_at FROM outbox_event WHERE event_id=?", knownID); err != nil {
		t.Fatal(err)
	}
	if known.Status != 3 || known.AttemptCount != 1 || !known.PublishedAt.Valid {
		t.Fatalf("known event state: %+v", known)
	}
	var unknown struct {
		Status       uint64 `db:"status"`
		AttemptCount uint64 `db:"attempt_count"`
	}
	if err := conn.QueryRowCtx(ctx, &unknown, "SELECT status,attempt_count FROM outbox_event WHERE event_id=?", unknownID); err != nil {
		t.Fatal(err)
	}
	if unknown.Status != 4 || unknown.AttemptCount != 1 {
		t.Fatalf("unknown event state: %+v", unknown)
	}
}

func TestDispatchFanOutRollsBackOnTaskInsertFailureMySQL(t *testing.T) {
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

	id := uuid.NewString()
	defer func() {
		_, _ = conn.ExecCtx(context.Background(), "DELETE FROM event_delivery WHERE event_id=?", id)
		_, _ = conn.ExecCtx(context.Background(), "DELETE FROM outbox_event WHERE event_id=?", id)
	}()
	if _, err := conn.ExecCtx(ctx, "INSERT INTO outbox_event(event_id,aggregate_type,aggregate_id,event_type,payload_json,occurred_at) VALUES (?,'Post',?,'PostPublished','{\"schemaVersion\":1}',NOW(3))", id, id); err != nil {
		t.Fatal(err)
	}
	if _, err := NewDispatchRepository(conn).FanOut(ctx, 1, duplicateConsumerResolver{}); err == nil {
		t.Fatal("duplicate task insertion unexpectedly succeeded")
	}
	var state struct {
		Status uint64 `db:"status"`
		Tasks  uint64 `db:"tasks"`
	}
	if err := conn.QueryRowCtx(ctx, &state, "SELECT o.status,(SELECT COUNT(*) FROM event_delivery d WHERE d.event_id=o.event_id) AS tasks FROM outbox_event o WHERE o.event_id=?", id); err != nil {
		t.Fatal(err)
	}
	if state.Status != 1 || state.Tasks != 0 {
		t.Fatalf("fan-out transaction was partially committed: %+v", state)
	}
}
