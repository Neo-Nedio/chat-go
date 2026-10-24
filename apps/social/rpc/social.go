package main

import (
	"chat-go/apps/social/rpc/internal/config"
	"chat-go/apps/social/rpc/internal/server"
	"chat-go/apps/social/rpc/internal/svc"
	"chat-go/apps/social/rpc/social"
	"chat-go/pkg/interceptor"
	"chat-go/pkg/interceptor/rpcserver"
	"flag"
	"fmt"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "apps/social/rpc/etc/dev/user.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		social.RegisterSocialServer(grpcServer, server.NewSocialServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	s.AddUnaryInterceptors(
		rpcserver.LogInterceptor,
		rpcserver.SyncLimiterInterceptor(10),
	)
	//✨ 在客户端传递参数即可，但在服务端需要传递出Redis的配置信息
	s.AddUnaryInterceptors(interceptor.NewIdempotenceServer(interceptor.NewDefaultIdempotent(c.Cache[0].RedisConf)))

	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
