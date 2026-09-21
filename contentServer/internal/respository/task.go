package respository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math/rand/v2"
	"time"
	"unicode/utf8"

	"contentserver/internal/event"
	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type TaskRepository struct{ conn sqlx.SqlConn }

func NewTaskRepository(conn sqlx.SqlConn) *TaskRepository { return &TaskRepository{conn: conn} }

var _ event.TaskRepository = (*TaskRepository)(nil)
var _ event.TaskClaimer = (*TaskRepository)(nil)

func (r *TaskRepository) Claim(ctx context.Context, limit int, lease time.Duration) ([]event.ClaimedTask, error) {
	if limit <= 0 {
		return nil, nil
	}
	if lease <= 0 {
		return nil, errors.New("task lease must be positive")
	}
	// Keep each claim transaction short even if a caller is misconfigured.
	if limit > 100 {
		limit = 100
	}

	claimed := make([]event.ClaimedTask, 0, limit)
	err := r.conn.TransactCtx(ctx, func(ctx context.Context, tx sqlx.Session) error {
		var rows []struct {
			EventID      string `db:"event_id"`
			ConsumerName string `db:"consumer_name"`
		}
		if err := tx.QueryRowsCtx(ctx, &rows, `
SELECT event_id, consumer_name
FROM event_delivery
WHERE status = 1
  AND next_attempt_at <= NOW(3)
  AND attempt_count < max_attempts
ORDER BY next_attempt_at, event_id, consumer_name
LIMIT ?
FOR UPDATE SKIP LOCKED`, limit); err != nil {
			return err
		}

		for _, row := range rows {
			token := uuid.NewString()
			result, err := tx.ExecCtx(ctx, `
UPDATE event_delivery
SET status = 2,
    attempt_count = attempt_count + 1,
    claim_token = ?,
    locked_until = TIMESTAMPADD(MICROSECOND, ?, NOW(3)),
    completed_at = NULL
WHERE event_id = ?
  AND consumer_name = ?
  AND status = 1
  AND next_attempt_at <= NOW(3)
  AND attempt_count < max_attempts`, token, lease.Microseconds(), row.EventID, row.ConsumerName)
			if err != nil {
				return err
			}
			affected, err := result.RowsAffected()
			if err != nil {
				return err
			}
			if affected != 1 {
				return errors.New("locked task could not be claimed")
			}
			var leaseRow struct {
				LockedUntil time.Time `db:"locked_until"`
			}
			if err := tx.QueryRowCtx(ctx, &leaseRow, `
SELECT locked_until
FROM event_delivery
WHERE event_id = ? AND consumer_name = ?`, row.EventID, row.ConsumerName); err != nil {
				return err
			}
			claimed = append(claimed, event.ClaimedTask{
				Context: event.TaskContext{
					EventId:      row.EventID,
					ConsumerName: row.ConsumerName,
				},
				ClaimToken:  token,
				LockedUntil: leaseRow.LockedUntil,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return claimed, nil
}

func (r *TaskRepository) Execute(ctx context.Context, task event.ClaimedTask, apply func(context.Context, sqlx.Session, *event.TaskContext) error) error {
	return r.conn.TransactCtx(ctx, func(ctx context.Context, tx sqlx.Session) error {
		var row struct {
			Status     uint64         `db:"status"`
			Token      sql.NullString `db:"claim_token"`
			LeaseValid bool           `db:"lease_valid"`
		}
		err := tx.QueryRowCtx(ctx, &row, "SELECT status,claim_token,COALESCE(locked_until>NOW(3),0) AS lease_valid FROM event_delivery WHERE event_id=? AND consumer_name=? FOR UPDATE", task.Context.EventId, task.Context.ConsumerName)
		if errors.Is(err, sqlx.ErrNotFound) {
			return event.ErrClaimLost
		}
		if err != nil {
			return err
		}
		if row.Status != 2 || !row.Token.Valid || row.Token.String != task.ClaimToken || !row.LeaseValid {
			return event.ErrClaimLost
		}
		var source struct {
			Type    string `db:"event_type"`
			Payload string `db:"payload_json"`
		}
		if err := tx.QueryRowCtx(ctx, &source, "SELECT event_type,payload_json FROM outbox_event WHERE event_id=?", task.Context.EventId); err != nil {
			return err
		}
		var header struct {
			SchemaVersion uint64 `json:"schemaVersion"`
		}
		if err := json.Unmarshal([]byte(source.Payload), &header); err != nil {
			return event.Permanent(err)
		}
		input := &event.TaskContext{EventId: task.Context.EventId, ConsumerName: task.Context.ConsumerName, EventType: source.Type, SchemaVersion: header.SchemaVersion, Payload: json.RawMessage(source.Payload)}
		err = apply(ctx, tx, input)
		status := 3
		if errors.Is(err, event.ErrSuperseded) {
			status = 4
		} else if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		// The task row is locked for this entire execution transaction. An expiry
		// reclaimer cannot transfer ownership while these business writes commit.
		result, err := tx.ExecCtx(ctx, "UPDATE event_delivery SET status=?,claim_token=NULL,locked_until=NULL,completed_at=NOW(3),last_error='' WHERE event_id=? AND consumer_name=? AND status=2 AND claim_token=?", status, input.EventId, input.ConsumerName, task.ClaimToken)
		if err != nil {
			return err
		}
		n, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if n != 1 {
			return event.ErrClaimLost
		}
		return nil
	})
}

func (r *TaskRepository) ScheduleRetry(ctx context.Context, task event.ClaimedTask, cause error, base time.Duration, permanent bool) error {
	return r.conn.TransactCtx(ctx, func(ctx context.Context, tx sqlx.Session) error {
		var row struct {
			Attempts uint64 `db:"attempt_count"`
			Max      uint64 `db:"max_attempts"`
		}
		err := tx.QueryRowCtx(ctx, &row, "SELECT attempt_count,max_attempts FROM event_delivery WHERE event_id=? AND consumer_name=? AND status=2 AND claim_token=? FOR UPDATE", task.Context.EventId, task.Context.ConsumerName, task.ClaimToken)
		if errors.Is(err, sqlx.ErrNotFound) {
			return event.ErrClaimLost
		}
		if err != nil {
			return err
		}
		// Bounded exponential backoff with jitter, capped at five minutes.
		if base <= 0 {
			base = time.Second
		}
		delay := base
		for i := uint64(1); i < row.Attempts && delay < 5*time.Minute; i++ {
			delay *= 2
		}
		if delay > 5*time.Minute {
			delay = 5 * time.Minute
		}
		delay = delay/2 + time.Duration(rand.Int64N(int64(delay/2)+1))
		message := ""
		if cause != nil {
			message = cause.Error()
		}
		if !utf8.ValidString(message) {
			message = string([]rune(message))
		}
		runes := []rune(message)
		if len(runes) > 1000 {
			message = string(runes[:1000])
		}
		if permanent || row.Attempts >= row.Max {
			_, err = tx.ExecCtx(ctx, "UPDATE event_delivery SET status=5,claim_token=NULL,locked_until=NULL,completed_at=NOW(3),last_error=? WHERE event_id=? AND consumer_name=? AND status=2 AND claim_token=?", message, task.Context.EventId, task.Context.ConsumerName, task.ClaimToken)
		} else {
			_, err = tx.ExecCtx(ctx, "UPDATE event_delivery SET status=1,claim_token=NULL,locked_until=NULL,completed_at=NULL,next_attempt_at=TIMESTAMPADD(MICROSECOND,?,NOW(3)),last_error=? WHERE event_id=? AND consumer_name=? AND status=2 AND claim_token=?", delay.Microseconds(), message, task.Context.EventId, task.Context.ConsumerName, task.ClaimToken)
		}
		return err
	})
}
