package svc

import (
	"chat-go/apps/im/rpc/imclient"
	"chat-go/apps/social/api/internal/config"
	"chat-go/apps/social/rpc/socialclient"
	"chat-go/apps/user/rpc/userclient"
	"chat-go/pkg/interceptor"
	"chat-go/pkg/middleware"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config config.Config
	*redis.Redis

	socialclient.Social
	userclient.User
	imclient.Im

	IdempotenceMiddleware rest.Middleware
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
		Redis:  redis.MustNewRedis(c.Redisx),
		//Social 客户端
		Social: socialclient.NewSocial(zrpc.MustNewClient(c.SocialRpc,
			// 重试 + 幂等
			//zrpc.WithDialOption(grpc.WithDefaultServiceConfig()),
			zrpc.WithUnaryClientInterceptor(interceptor.DefaultIdempotentClient),
		)),
		User:                  userclient.NewUser(zrpc.MustNewClient(c.UserRpc)),
		Im:                    imclient.NewIm(zrpc.MustNewClient(c.ImRpc)),
		IdempotenceMiddleware: middleware.NewIdempotenceMiddleware().Handler,
	}
}
