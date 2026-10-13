package rpcserver

import (
	"context"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
	zerr "github.com/zeromicro/x/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func LogInterceptor(ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (resp any, err error) {

	resp, err = handler(ctx, req)
	if err == nil {
		return resp, nil
	}

	// 先记原始错误，带方法名
	logx.WithContext(ctx).Errorf("【RPC SRV ERR】method=%s err=%v", info.FullMethod, err)

	// 业务错误 → 转 gRPC status
	var e *zerr.CodeMsg
	if errors.As(err, &e) {
		return nil, status.Error(codes.Code(e.Code), e.Msg)
	}

	// 非业务错误 → 通用错误，不暴露内部细节
	return nil, status.Error(codes.Internal, "internal error")
}
