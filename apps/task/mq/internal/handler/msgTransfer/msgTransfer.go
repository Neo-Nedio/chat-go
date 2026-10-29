package msgTransfer

import (
	"chat-go/apps/im/ws/websocket"
	"chat-go/apps/im/ws/ws"
	"chat-go/apps/social/rpc/socialclient"
	"chat-go/apps/task/mq/internal/svc"
	"chat-go/pkg/constants"
	"context"

	"github.com/zeromicro/go-zero/core/logx"
)

type baseMsgTransfer struct {
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewBaseMsgTransfer(svc *svc.ServiceContext) *baseMsgTransfer {
	return &baseMsgTransfer{
		svcCtx: svc,
		Logger: logx.WithContext(context.Background()),
	}
}

func (m *baseMsgTransfer) Transfer(ctx context.Context, data *ws.Push) error {
	var err error
	switch data.ChatType {
	case constants.SingleChatType:
		err = m.single(ctx, data)
	case constants.GroupChatType:
		err = m.group(ctx, data)
	}
	return err
}

func (m *baseMsgTransfer) single(ctx context.Context, data *ws.Push) error {
	return m.svcCtx.WsClient.Send(websocket.Message{
		FrameType: websocket.FrameData,
		Method:    "push",
		Id:        "",
		FormId:    constants.SYSTEM_ROOT_UID,
		Data:      data,
	})
}

func (m *baseMsgTransfer) group(ctx context.Context, data *ws.Push) error {
	// 查询群的用户
	users, err := m.svcCtx.Social.GroupUsers(ctx, &socialclient.GroupUsersReq{
		GroupId: data.RecvId,
	})
	if err != nil {
		return err
	}
	data.RecvIds = make([]string, 0, len(users.List))
	//fmt.Printf("group user: %+v", users.List)
	for _, members := range users.List {
		if members.UserId == data.SendId {
			continue
		}
		data.RecvIds = append(data.RecvIds, members.UserId)
	}
	return m.svcCtx.WsClient.Send(websocket.Message{
		FrameType: websocket.FrameData,
		Method:    "push",
		Id:        "",
		FormId:    constants.SYSTEM_ROOT_UID,
		Data:      data,
	})

}
