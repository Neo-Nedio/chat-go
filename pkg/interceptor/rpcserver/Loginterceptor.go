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
	//先调真正的业务逻辑，拿到结果和错误
	resp, err = handler(ctx, req)
	if err == nil {
		return resp, err
	}

	//取出最内层的原始错误。因为你务里可能用 errors.Wrapf 包装过，Cause 能剥掉包装，拿到 xerr 定义的原始错误。
	causeErr := errors.Cause(err)
	var e *zerr.CodeMsg
	if errors.As(causeErr, &e) {
		err = status.Error(codes.Code(e.Code), e.Msg)
	}

	logx.WithContext(ctx).Errorf("『 RPC SRV ERR 』%v", err)
	return
}
