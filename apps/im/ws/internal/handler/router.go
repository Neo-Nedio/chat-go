package handler

import (
	"chat-go/apps/im/ws/internal/handler/conversation"
	"chat-go/apps/im/ws/internal/handler/push"
	"chat-go/apps/im/ws/internal/handler/user"
	"chat-go/apps/im/ws/internal/svc"
	"chat-go/apps/im/ws/websocket"
)

// 与api层不同，这里注册的是我们自定义的websocket服务
func RegisterHandlers(srv *websocket.Server, svc *svc.ServiceContext) {
	srv.AddRoutes([]websocket.Route{
		{
			Method:  "user.online",
			Handler: user.OnLine(svc),
		},
		{
			Method:  "conversation.chat",
			Handler: conversation.Chat(svc),
		},
		{
			Method:  "push",
			Handler: push.Push(svc),
		},
	})
}
