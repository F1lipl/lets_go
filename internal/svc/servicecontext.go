// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package svc

import (
	"userServer/internal/config"
	"userServer/internal/model"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config          config.Config
	Redis           *redis.Redis
	UserModel       model.UsersModel
	UserDeviceModel model.UserDevicesModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	mysqlConn := sqlx.NewMysql(c.DataSource)
	return &ServiceContext{
		Config:          c,
		Redis:           redis.MustNewRedis(c.Redis),
		UserModel:       model.NewUsersModel(mysqlConn),
		UserDeviceModel: model.NewUserDevicesModel(mysqlConn),
	}
}
