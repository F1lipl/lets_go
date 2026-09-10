// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package svc

import (
	"contentserver/internal/config"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config config.Config
	DB     sqlx.SqlConn
}

func NewServiceContext(c config.Config) *ServiceContext {
	db := sqlx.NewMysql(c.DataSource)

	return &ServiceContext{
		Config: c,
		DB:     db,
	}
}
