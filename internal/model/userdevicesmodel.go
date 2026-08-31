package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ UserDevicesModel = (*customUserDevicesModel)(nil)

type (
	// UserDevicesModel is an interface to be customized, add more methods here,
	// and implement the added methods in customUserDevicesModel.
	UserDevicesModel interface {
		userDevicesModel
		withSession(session sqlx.Session) UserDevicesModel
	}

	customUserDevicesModel struct {
		*defaultUserDevicesModel
	}
)

// NewUserDevicesModel returns a model for the database table.
func NewUserDevicesModel(conn sqlx.SqlConn) UserDevicesModel {
	return &customUserDevicesModel{
		defaultUserDevicesModel: newUserDevicesModel(conn),
	}
}

func (m *customUserDevicesModel) withSession(session sqlx.Session) UserDevicesModel {
	return NewUserDevicesModel(sqlx.NewSqlConnFromSession(session))
}
