package middleware

import (
	"chat-go/pkg/constants"
	"net/http"

	"github.com/zeromicro/go-zero/core/limit"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
)

// LimitMiddleware HTTP 限流中间件，用 go-zero 的令牌桶算法限制请求速率：
// 每秒生成固定数量的令牌，请求要拿到令牌才能继续，拿不到就返回 429 Too Many Requests
type LimitMiddleware struct {
	redisCfg redis.RedisConf // Redis 配置
	rds      *redis.Redis    // 复用的 Redis 客户端，避免每次创建
}

func NewLimitMiddleware(redisCfg redis.RedisConf) *LimitMiddleware {
	return &LimitMiddleware{
		redisCfg: redisCfg,
		rds:      redis.MustNewRedis(redisCfg),
	}
}

// TokenLimitHandler @Param 限流速率(每秒生成多少令牌), 桶容量
func (m *LimitMiddleware) TokenLimitHandler(name string, rate, burst int) rest.Middleware {
	// 创建令牌桶限流器
	/*
		rate：每秒生成多少令牌，也就是长期平均允许的 QPS。
		burst：桶容量，允许的突发请求数（桶满时最多攒这么多令牌）。
		m.rds：令牌桶状态存在 Redis 里，多个服务实例共享同一个桶，所以是分布式限流。
		constants.RedisTokenLimitKey：Redis 里存令牌桶的 key。
	*/

	// 内部拼接完整的 Redis key
	key := constants.RedisTokenLimitKey + ":" + name
	limiter := limit.NewTokenLimiter(rate, burst, m.rds, key)

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// 如果获取令牌成功，继续执行业务
			if limiter.AllowCtx(r.Context()) {
				next(w, r)
				return
			}
			// 限流（抛出指定状态码，go-zero 限流源码自动识别并执行限流）
			w.WriteHeader(http.StatusTooManyRequests)
		}
	}
}
