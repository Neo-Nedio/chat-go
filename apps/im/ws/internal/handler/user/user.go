package user

import (
	"chat-go/apps/im/ws/internal/svc"
	"chat-go/apps/im/ws/websocket"
)

func OnLine(svcCtx *svc.ServiceContext) websocket.HandlerFunc {
	return func(srv *websocket.Server, conn *websocket.Conn, msg *websocket.Message) {
		ids := srv.GetUsers()
		u := srv.GetUsers(conn)
		if len(u) == 0 {
			return
		}
		if err := srv.Send(websocket.NewMessage(u[0], ids), conn); err != nil {
			srv.Error(err)
		}
	}
}
