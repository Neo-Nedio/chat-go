package svc

import (
	"chat-go/apps/im/api/internal/config"
	"chat-go/apps/im/rpc/im"
	"chat-go/apps/im/rpc/imclient"
	"chat-go/apps/social/rpc/social"
	"chat-go/apps/social/rpc/socialclient"
	"chat-go/apps/user/rpc/user"
	"chat-go/apps/user/rpc/userclient"

	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config config.Config

	im.ImClient
	social.SocialClient
	user.UserClient
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:       c,
		ImClient:     imclient.NewIm(zrpc.MustNewClient(c.ImRpc)),
		SocialClient: socialclient.NewSocial(zrpc.MustNewClient(c.SocialRpc)),
		UserClient:   userclient.NewUser(zrpc.MustNewClient(c.UserRpc)),
	}
}
