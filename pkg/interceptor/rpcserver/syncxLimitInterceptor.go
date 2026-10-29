package rpcserver

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/syncx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SyncLimiterInterceptor gRPC 服务端并发限流拦截器：用 go-zero 的 syncx.Limit 控制同时进入业务 handler 的请求数量
// 超过上限就直接拒绝，返回 Unavailable 错误码
func SyncLimiterInterceptor(maxCount int) grpc.UnaryServerInterceptor {
	//创建并发限制器
	l := syncx.NewLimit(maxCount) // 定义最大并发数量
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		if l.TryBorrow() { // 判断当前是否进行了限流
			//成功（当前并发 < maxCount）→ 进入 if，defer l.Return() 保证 handler 结束后归还名额。
			defer func() {
				if err := l.Return(); err != nil {
					logx.Error(err)
				}
			}()
		} else {
			//失败（已达上限）→ 走 else，直接拒绝，返回 codes.Unavailable
			logx.Errorf("concurrent connections over %d,rejected with code %d",
				maxCount, http.StatusServiceUnavailable)
			return nil, status.Error(codes.Unavailable, "concurrent connection over limit")
		}
		resp, err = handler(ctx, req)
		return
	}
}
