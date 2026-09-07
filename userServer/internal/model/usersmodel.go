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
		FindOneForUpdate(ctx context.Context, userID string) (*Users, error)
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

func (m *customUsersModel) FindOneForUpdate(ctx context.Context, userID string) (*Users, error) {
	query := fmt.Sprintf(
		"select %s from %s where `user_id` = ? limit 1 for update",
		usersRows,
		m.table,
	)

	var resp Users
	if err := m.conn.QueryRowCtx(ctx, &resp, query, userID); err != nil {
		if err == sqlx.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &resp, nil
}
