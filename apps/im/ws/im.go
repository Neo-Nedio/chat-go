package main

import (
	"chat-go/apps/im/ws/internal/config"
	"chat-go/apps/im/ws/internal/handler"
	"chat-go/apps/im/ws/internal/svc"
	"chat-go/apps/im/ws/websocket"
	"flag"
	"fmt"

	"github.com/zeromicro/go-zero/core/conf"
)

var configFile = flag.String("f", "etc/dev/im.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	if err := c.SetUp(); err != nil {
		panic(err)
	}

	svc.NewServiceContext(c)

	srv := websocket.NewServer(c.ListenOn)
	defer srv.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(srv, ctx)

	fmt.Println("start websocket server at ", c.ListenOn, " ..... ")
	srv.Start()
}
