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
	"contentserver/internal/model"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type TaskRepository struct{ conn sqlx.SqlConn }

func NewTaskRepository(conn sqlx.SqlConn) *TaskRepository { return &TaskRepository{conn: conn} }

var _ event.TaskRepository = (*TaskRepository)(nil)
var _ event.TaskClaimer = (*TaskRepository)(nil)
var _ event.TaskRecoverer = (*TaskRepository)(nil)

const expiredTaskError = "task processing lease expired"

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
		txConn := sqlx.NewSqlConnFromSession(tx)
		deliveryModel := model.NewEventDeliveryModel(txConn)
		rows, err := deliveryModel.FindReadyForUpdate(ctx, limit)
		if err != nil {
			return err
		}

		for _, row := range rows {
			token := uuid.NewString()
			lockedUntil, err := deliveryModel.MarkProcessing(ctx, row.EventId, row.ConsumerName, token, lease)
			if errors.Is(err, model.ErrNotFound) {
				return errors.New("locked task could not be claimed")
			}
			if err != nil {
				return err
			}
			claimed = append(claimed, event.ClaimedTask{
				Context: event.TaskContext{
					EventId:      row.EventId,
					ConsumerName: row.ConsumerName,
				},
				ClaimToken:  token,
				LockedUntil: lockedUntil,
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
		txConn := sqlx.NewSqlConnFromSession(tx)
		deliveryModel := model.NewEventDeliveryModel(txConn)
		row, err := deliveryModel.FindClaimStateForUpdate(ctx, task.Context.EventId, task.Context.ConsumerName)
		if errors.Is(err, model.ErrNotFound) {
			return event.ErrClaimLost
		}
		if err != nil {
			return err
		}
		if row.Status != 2 || !row.ClaimToken.Valid || row.ClaimToken.String != task.ClaimToken || !row.LeaseValid {
			return event.ErrClaimLost
		}
		outboxModel := model.NewOutboxEventModel(txConn)
		source, err := outboxModel.FindOne(ctx, task.Context.EventId)
		if err != nil {
			return err
		}
		var header struct {
			SchemaVersion uint64 `json:"schemaVersion"`
		}
		if err := json.Unmarshal([]byte(source.PayloadJson), &header); err != nil {
			return event.Permanent(err)
		}
		input := &event.TaskContext{EventId: task.Context.EventId, ConsumerName: task.Context.ConsumerName, EventType: source.EventType, SchemaVersion: header.SchemaVersion, Payload: json.RawMessage(source.PayloadJson)}
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
		result, err := deliveryModel.MarkTerminal(ctx, input.EventId, input.ConsumerName, task.ClaimToken, uint64(status))
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
		txConn := sqlx.NewSqlConnFromSession(tx)
		deliveryModel := model.NewEventDeliveryModel(txConn)
		row, err := deliveryModel.FindRetryStateForUpdate(ctx, task.Context.EventId, task.Context.ConsumerName, task.ClaimToken)
		if errors.Is(err, model.ErrNotFound) {
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
		for i := uint64(1); i < row.AttemptCount && delay < 5*time.Minute; i++ {
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
		var result sql.Result
		if permanent || row.AttemptCount >= row.MaxAttempts {
			result, err = deliveryModel.MarkFailed(ctx, task.Context.EventId, task.Context.ConsumerName, task.ClaimToken, message)
		} else {
			result, err = deliveryModel.MarkPending(ctx, task.Context.EventId, task.Context.ConsumerName, task.ClaimToken, message, delay)
		}
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected != 1 {
			return event.ErrClaimLost
		}
		return nil
	})
}

func (r *TaskRepository) RecoverExpired(ctx context.Context, limit int) (event.RecoveryResult, error) {
	if limit <= 0 {
		return event.RecoveryResult{}, nil
	}
	if limit > 500 {
		limit = 500
	}

	var recovered event.RecoveryResult
	err := r.conn.TransactCtx(ctx, func(ctx context.Context, tx sqlx.Session) error {
		deliveryModel := model.NewEventDeliveryModel(sqlx.NewSqlConnFromSession(tx))
		rows, err := deliveryModel.FindExpiredForUpdate(ctx, limit)
		if err != nil {
			return err
		}
		recovered.Scanned = len(rows)
		for _, row := range rows {
			var result sql.Result
			if row.AttemptCount >= row.MaxAttempts {
				result, err = deliveryModel.MarkExpiredFailed(ctx, row.EventId, row.ConsumerName, row.ClaimToken, expiredTaskError)
			} else {
				result, err = deliveryModel.MarkExpiredPending(ctx, row.EventId, row.ConsumerName, row.ClaimToken, expiredTaskError)
			}
			if err != nil {
				return err
			}
			affected, err := result.RowsAffected()
			if err != nil {
				return err
			}
			if affected != 1 {
				return errors.New("expired task recovery lost its claim")
			}
			if row.AttemptCount >= row.MaxAttempts {
				recovered.Failed++
			} else {
				recovered.Recovered++
			}
		}
		return nil
	})
	if err != nil {
		return event.RecoveryResult{}, err
	}
	return recovered, nil
}
