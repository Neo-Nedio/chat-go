// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package svc

import (
	"chat-go/apps/user/api/internal/config"
	"chat-go/apps/user/rpc/userclient"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config config.Config

	userclient.User
	*redis.Redis
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,

		User:  userclient.NewUser(zrpc.MustNewClient(c.UserRpc)),
		Redis: redis.MustNewRedis(c.Redisx),
	}
}
