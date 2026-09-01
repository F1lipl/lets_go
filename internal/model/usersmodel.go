package model

import (
	"context"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ UsersModel = (*customUsersModel)(nil)

type (
	// UsersModel is an interface to be customized, add more methods here,
	// and implement the added methods in customUsersModel.
	UsersModel interface {
		usersModel
		UpdateLastLoginAt(
			ctx context.Context,
			userID string,
			lastLoginAt time.Time,
		) error
		withSession(session sqlx.Session) UsersModel
	}

	customUsersModel struct {
		*defaultUsersModel
	}
)

// NewUsersModel returns a model for the database table.
func NewUsersModel(conn sqlx.SqlConn) UsersModel {
	return &customUsersModel{
		defaultUsersModel: newUsersModel(conn),
	}
}

func (m *customUsersModel) withSession(session sqlx.Session) UsersModel {
	return NewUsersModel(sqlx.NewSqlConnFromSession(session))
}

func (m *customUsersModel) UpdateLastLoginAt(
	ctx context.Context,
	userID string,
	lastLoginAt time.Time,
) error {
	query := fmt.Sprintf(
		"update %s set `last_login_at` = ? where `user_id` = ?",
		m.table,
	)

	_, err := m.conn.ExecCtx(
		ctx,
		query,
		lastLoginAt,
		userID,
	)

	return err
}
