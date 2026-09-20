package model

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// Verifies database invariants only; dispatcher/worker implementation is separate.
func TestEventDeliverySchemaMySQL(t *testing.T) {
	dsn := os.Getenv("CONTENT_MODEL_TEST_DSN")
	if dsn == "" {
		t.Skip("set CONTENT_MODEL_TEST_DSN to test MySQL")
	}
	// Own the connection pool so other tests closing their pool cannot affect this test.
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	conn := sqlx.NewSqlConnFromDB(db)
	ctx := context.Background()
	rollback := errors.New("rollback event delivery fixtures")
	err = conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		tx := sqlx.NewSqlConnFromSession(session)
		id := uuid.NewString()
		if _, err := tx.ExecCtx(ctx, "INSERT INTO outbox_event (event_id,aggregate_type,aggregate_id,event_type,payload_json,occurred_at) VALUES (?,'Post',?,'PostPublished','{}',NOW(3))", id, id); err != nil {
			return err
		}
		if _, err := tx.ExecCtx(ctx, "INSERT INTO event_delivery (event_id,consumer_name) VALUES (?,'card'),(?,'tag')", id, id); err != nil {
			return err
		}
		if _, err := tx.ExecCtx(ctx, "INSERT INTO event_delivery (event_id,consumer_name) VALUES (?,'card')", id); err == nil {
			t.Error("duplicate delivery accepted")
		}
		if _, err := tx.ExecCtx(ctx, "INSERT INTO event_delivery (event_id,consumer_name) VALUES (?,'card')", uuid.NewString()); err == nil {
			t.Error("delivery without event accepted")
		}
		if _, err := tx.ExecCtx(ctx, "UPDATE event_delivery SET status=2 WHERE event_id=? AND consumer_name='card'", id); err == nil {
			t.Error("processing without lease accepted")
		}
		if _, err := tx.ExecCtx(ctx, "UPDATE event_delivery SET status=3 WHERE event_id=? AND consumer_name='card'", id); err == nil {
			t.Error("terminal status without completion time accepted")
		}
		token := uuid.NewString()
		if _, err := tx.ExecCtx(ctx, "UPDATE event_delivery SET status=2,claim_token=?,locked_until=TIMESTAMPADD(SECOND,30,NOW(3)),attempt_count=attempt_count+1 WHERE event_id=? AND consumer_name='card' AND status=1", token, id); err != nil {
			return err
		}
		result, err := tx.ExecCtx(ctx, "UPDATE event_delivery SET status=3,claim_token=NULL,locked_until=NULL,completed_at=NOW(3) WHERE event_id=? AND consumer_name='card' AND status=2 AND claim_token=?", id, uuid.NewString())
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected != 0 {
			t.Error("stale claim changed task state")
		}
		if _, err := tx.ExecCtx(ctx, "UPDATE event_delivery SET status=4,claim_token=NULL,locked_until=NULL,completed_at=NOW(3) WHERE event_id=? AND consumer_name='card' AND claim_token=?", id, token); err != nil {
			return err
		}
		if _, err := tx.ExecCtx(ctx, "UPDATE event_delivery SET status=3,completed_at=NOW(3) WHERE event_id=? AND consumer_name='tag'", id); err != nil {
			return err
		}
		var outstanding int
		if err := tx.QueryRowCtx(ctx, &outstanding, "SELECT COUNT(*) FROM event_delivery WHERE event_id=? AND status NOT IN (3,4)", id); err != nil {
			return err
		}
		if outstanding != 0 {
			t.Error("superseded counted as outstanding")
		}
		if _, err := tx.ExecCtx(ctx, "INSERT INTO inbox_event (consumer_name,event_id,event_type,status,payload_json,processed_at) VALUES ('card',?,'PostPublished',4,'{}',NOW(3))", id); err != nil {
			return err
		}
		if _, err := tx.ExecCtx(ctx, "DELETE FROM outbox_event WHERE event_id=?", id); err == nil {
			t.Error("parent deleted while deliveries remain")
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatal(err)
	}
}
