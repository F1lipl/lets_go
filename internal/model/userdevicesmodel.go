package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ UserDevicesModel = (*customUserDevicesModel)(nil)

type (
	UserDevicesModel interface {
		userDevicesModel
		withSession(session sqlx.Session) UserDevicesModel
	}

	customUserDevicesModel struct {
		*defaultUserDevicesModel
	}
)

func NewUserDevicesModel(conn sqlx.SqlConn) UserDevicesModel {
	return &customUserDevicesModel{
		defaultUserDevicesModel: newUserDevicesModel(conn),
	}
}

func (m *customUserDevicesModel) withSession(
	session sqlx.Session,
) UserDevicesModel {
	return NewUserDevicesModel(
		sqlx.NewSqlConnFromSession(session),
	)
}
