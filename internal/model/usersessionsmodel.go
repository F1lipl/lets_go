package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ UserSessionsModel = (*customUserSessionsModel)(nil)

type (
	// UserSessionsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customUserSessionsModel.
	UserSessionsModel interface {
		userSessionsModel
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
