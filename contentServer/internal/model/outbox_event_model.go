package model

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ OutboxEventModel = (*customOutboxEventModel)(nil)

type (
	// OutboxEventModel is an interface to be customized, add more methods here,
	// and implement the added methods in customOutboxEventModel.
	OutboxEventModel interface {
		outboxEventModel
		withSession(session sqlx.Session) OutboxEventModel
		FindPendingForDispatch(ctx context.Context, limit int) ([]OutboxDispatchKey, error)
		MarkDispatched(ctx context.Context, eventId string) (sql.Result, error)
		MarkDispatchFailed(ctx context.Context, eventId string) (sql.Result, error)
	}

	customOutboxEventModel struct {
		*defaultOutboxEventModel
	}
)

type OutboxDispatchKey struct {
	EventId   string `db:"event_id"`
	EventType string `db:"event_type"`
}

// NewOutboxEventModel returns a model for the database table.
func NewOutboxEventModel(conn sqlx.SqlConn) OutboxEventModel {
	return &customOutboxEventModel{
		defaultOutboxEventModel: newOutboxEventModel(conn),
	}
}

func (m *customOutboxEventModel) withSession(session sqlx.Session) OutboxEventModel {
	return NewOutboxEventModel(sqlx.NewSqlConnFromSession(session))
}

func (m *customOutboxEventModel) FindPendingForDispatch(ctx context.Context, limit int) ([]OutboxDispatchKey, error) {
	query := fmt.Sprintf(`
SELECT event_id, event_type
FROM %s
WHERE status = 1
  AND next_attempt_at <= NOW(3)
ORDER BY next_attempt_at, event_id
LIMIT ?
FOR UPDATE SKIP LOCKED`, m.table)
	var rows []OutboxDispatchKey
	if err := m.conn.QueryRowsCtx(ctx, &rows, query, limit); err != nil {
		return nil, err
	}
	return rows, nil
}

func (m *customOutboxEventModel) MarkDispatched(ctx context.Context, eventId string) (sql.Result, error) {
	query := fmt.Sprintf("UPDATE %s SET status=3,attempt_count=attempt_count+1,published_at=NOW(3) WHERE event_id=? AND status=1", m.table)
	return m.conn.ExecCtx(ctx, query, eventId)
}

func (m *customOutboxEventModel) MarkDispatchFailed(ctx context.Context, eventId string) (sql.Result, error) {
	query := fmt.Sprintf("UPDATE %s SET status=4,attempt_count=attempt_count+1,published_at=NULL WHERE event_id=? AND status=1", m.table)
	return m.conn.ExecCtx(ctx, query, eventId)
}
