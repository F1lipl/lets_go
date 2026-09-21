package respository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"contentserver/internal/event"
	"contentserver/internal/model"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type DispatchRepository struct {
	conn sqlx.SqlConn
}

func requireOneRow(result sql.Result) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return errors.New("expected one affected row")
	}
	return nil
}

func NewDispatchRepository(conn sqlx.SqlConn) *DispatchRepository {
	return &DispatchRepository{conn: conn}
}

var _ event.DispatchRepository = (*DispatchRepository)(nil)

func (r *DispatchRepository) FanOut(ctx context.Context, limit int, resolver event.ConsumerResolver) (event.DispatchResult, error) {
	if limit <= 0 {
		return event.DispatchResult{}, nil
	}
	if resolver == nil {
		return event.DispatchResult{}, errors.New("consumer resolver is required")
	}
	if limit > 500 {
		limit = 500
	}

	var result event.DispatchResult
	err := r.conn.TransactCtx(ctx, func(ctx context.Context, tx sqlx.Session) error {
		txConn := sqlx.NewSqlConnFromSession(tx)
		outboxModel := model.NewOutboxEventModel(txConn)
		deliveryModel := model.NewEventDeliveryModel(txConn)
		events, err := outboxModel.FindPendingForDispatch(ctx, limit)
		if err != nil {
			return err
		}
		result.ScannedEvents = len(events)

		for _, source := range events {
			consumers, ok := resolver.Consumers(source.EventType)
			if !ok || len(consumers) == 0 {
				updated, err := outboxModel.MarkDispatchFailed(ctx, source.EventId)
				if err != nil {
					return err
				}
				if err := requireOneRow(updated); err != nil {
					return fmt.Errorf("mark outbox event %s failed: %w", source.EventId, err)
				}
				result.FailedEvents++
				continue
			}

			for _, consumer := range consumers {
				if _, err := deliveryModel.InsertPending(ctx, source.EventId, consumer); err != nil {
					return err
				}
				result.CreatedTasks++
			}
			updated, err := outboxModel.MarkDispatched(ctx, source.EventId)
			if err != nil {
				return err
			}
			if err := requireOneRow(updated); err != nil {
				return fmt.Errorf("mark outbox event %s dispatched: %w", source.EventId, err)
			}
			result.DispatchedEvents++
		}
		return nil
	})
	if err != nil {
		return event.DispatchResult{}, err
	}
	return result, nil
}
