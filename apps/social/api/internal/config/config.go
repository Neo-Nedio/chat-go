package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	rest.RestConf

	SocialRpc zrpc.RpcClientConf
	UserRpc   zrpc.RpcClientConf
	ImRpc     zrpc.RpcClientConf // ← 嵌入 imclient.Im 接口

	Redisx redis.RedisConf

	JwtAuth struct {
		AccessSecret string
		//AccessExpire int64
	}
}
