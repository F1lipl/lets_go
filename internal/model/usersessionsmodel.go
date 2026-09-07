package model

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ UserSessionsModel = (*customUserSessionsModel)(nil)

type (
	// UserSessionsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customUserSessionsModel.
	UserSessionsModel interface {
		userSessionsModel
		FindOneForUpdate(ctx context.Context, id string) (*UserSessions, error)
		withSession(session sqlx.Session) UserSessionsModel
	}

	customUserSessionsModel struct {
		*defaultUserSessionsModel
	}
)

// NewUserSessionsModel returns a model for the database table.
func NewUserSessionsModel(conn sqlx.SqlConn) UserSessionsModel {
	return &customUserSessionsModel{
		defaultUserSessionsModel: newUserSessionsModel(conn),
	}
}

func (m *customUserSessionsModel) withSession(session sqlx.Session) UserSessionsModel {
	return NewUserSessionsModel(sqlx.NewSqlConnFromSession(session))
}

// FindOneForUpdate reads and locks one session until the current transaction ends.
func (m *customUserSessionsModel) FindOneForUpdate(
	ctx context.Context,
	id string,
) (*UserSessions, error) {
	query := fmt.Sprintf(
		"select %s from %s where `id` = ? limit 1 for update",
		userSessionsRows,
		m.table,
	)

	var resp UserSessions
	if err := m.conn.QueryRowCtx(ctx, &resp, query, id); err != nil {
		if err == sqlx.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &resp, nil
}
