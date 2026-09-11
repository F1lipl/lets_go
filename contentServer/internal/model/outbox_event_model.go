package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ OutboxEventModel = (*customOutboxEventModel)(nil)

type (
	// OutboxEventModel is an interface to be customized, add more methods here,
	// and implement the added methods in customOutboxEventModel.
	OutboxEventModel interface {
		outboxEventModel
		withSession(session sqlx.Session) OutboxEventModel
	}

	customOutboxEventModel struct {
		*defaultOutboxEventModel
	}
)

// NewOutboxEventModel returns a model for the database table.
func NewOutboxEventModel(conn sqlx.SqlConn) OutboxEventModel {
	return &customOutboxEventModel{
		defaultOutboxEventModel: newOutboxEventModel(conn),
	}
}

func (m *customOutboxEventModel) withSession(session sqlx.Session) OutboxEventModel {
	return NewOutboxEventModel(sqlx.NewSqlConnFromSession(session))
}
