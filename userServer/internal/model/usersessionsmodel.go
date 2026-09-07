package model

import (
	"context"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ UserSessionsModel = (*customUserSessionsModel)(nil)

type (
	// UserSessionsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customUserSessionsModel.
	UserSessionsModel interface {
		userSessionsModel
		FindOneForUpdate(ctx context.Context, id string) (*UserSessions, error)
		RevokeOtherByUserID(
			ctx context.Context,
			userID string,
			currentSessionID string,
			revokedAt time.Time,
			reason string,
		) error
		RevokeAllByUserID(ctx context.Context, userID string, revokeAt time.Time, reason string) error
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
func (m *customUserSessionsModel) RevokeOtherByUserID(
	ctx context.Context,
	userID string,
	currentSessionID string,
	revokedAt time.Time,
	reason string,
) error {
	query := fmt.Sprintf(`
		update %s
		set status = 0,
			revoked_at = ?,
			revoke_reason = ?
		where user_id = ?
			and id <> ?
			and status = 1
	`, m.table)

	_, err := m.conn.ExecCtx(
		ctx,
		query,
		revokedAt,
		reason,
		userID,
		currentSessionID,
	)
	return err
}

func (m *customUserSessionsModel) RevokeAllByUserID(
	ctx context.Context,
	userID string,
	revokedAt time.Time,
	reason string,
) error {
	query := fmt.Sprintf(`
		update %s
		set status = 0,
			revoked_at = ?,
			revoke_reason = ?
		where user_id = ?
			and status = 1
	`, m.table)

	_, err := m.conn.ExecCtx(
		ctx,
		query,
		revokedAt,
		reason,
		userID,
	)
	return err
}
