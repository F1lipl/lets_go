// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package svc

import (
	"userServer/internal/config"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

type ServiceContext struct {
	Config config.Config
	Redis  *redis.Redis
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
		Redis:  redis.MustNewRedis(c.Redis),
	}
}
