package model

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ UserDevicesModel = (*customUserDevicesModel)(nil)

type (
	UserDevicesModel interface {
		userDevicesModel

		UpsertLogin(
			ctx context.Context,
			data *UserDevices,
		) error

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

func (m *customUserDevicesModel) UpsertLogin(
	ctx context.Context,
	data *UserDevices,
) error {
	query := fmt.Sprintf(`
INSERT INTO %s (
    id,
    user_id,
    client_instance_id,
    device_name,
    platform,
    app_version,
    first_login_at,
    last_login_at
) VALUES (
    ?, ?, ?, ?, ?, ?,
    CURRENT_TIMESTAMP(3),
    CURRENT_TIMESTAMP(3)
)
ON DUPLICATE KEY UPDATE
    device_name = ?,
    platform = ?,
    app_version = ?,
    last_login_at = CURRENT_TIMESTAMP(3)
`, m.table)

	_, err := m.conn.ExecCtx(
		ctx,
		query,
		// INSERT 参数
		data.Id,
		data.UserId,
		data.ClientInstanceId,
		data.DeviceName,
		data.Platform,
		data.AppVersion,

		// UPDATE 参数
		data.DeviceName,
		data.Platform,
		data.AppVersion,
	)

	return err
}

func (m *customUserDevicesModel) withSession(
	session sqlx.Session,
) UserDevicesModel {
	return NewUserDevicesModel(
		sqlx.NewSqlConnFromSession(session),
	)
}
