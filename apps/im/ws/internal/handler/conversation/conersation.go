package conversation

import (
	"chat-go/apps/im/ws/internal/svc"
	"chat-go/apps/im/ws/websocket"
	"chat-go/apps/im/ws/ws"
	"chat-go/apps/task/mq/mq"
	"chat-go/pkg/constants"
	"time"

	"github.com/mitchellh/mapstructure"
)

// Chat 单独定义会话是因为聊天本身是需要在会话中去完成的，所以不在用户中去创建会话
func Chat(svc *svc.ServiceContext) websocket.HandlerFunc {
	return func(srv *websocket.Server, conn *websocket.Conn, msg *websocket.Message) {
		// todo: 私聊
		var data ws.Chat
		// 对map[string]interface{}类型进行转换
		if err := mapstructure.Decode(msg.Data, &data); err != nil {
			srv.Send(websocket.NewErrMessage(err), conn)
			return
		}
		switch data.ChatType {
		case constants.SingleChatType:
			//交给kafka，让他记录消息到数据库，并且重新调用wsClient发送消息
			err := svc.MsgChatTransferClient.Push(&mq.MsgChatTransfer{
				ConversationId: data.ConversationId,
				ChatType:       data.ChatType,
				SendId:         conn.Uid,
				RecvId:         data.RecvId,
				SendTime:       time.Now().UnixNano(),
				MType:          data.Msg.MType,
				Content:        data.Msg.Content,
			})
			if err != nil {
				srv.Send(websocket.NewErrMessage(err), conn)
				return
			}
		}
	}
}
