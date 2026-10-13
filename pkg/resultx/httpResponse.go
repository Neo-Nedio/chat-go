package resultx

import (
	"chat-go/pkg/xerr"
	"context"
	"net/http"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
	zrpcErr "github.com/zeromicro/x/errors"
	"google.golang.org/grpc/status"
)

type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

func Success(data interface{}) *Response {
	return &Response{
		Code: 200,
		Msg:  "",
		Data: data,
	}
}

func Fail(code int, err string) *Response {
	return &Response{
		Code: code,
		Msg:  err,
		Data: nil,
	}
}

func OkHandler(_ context.Context, v interface{}) any {
	return Success(v)
}

func ErrHandler(name string) func(ctx context.Context, err error) (int, any) {
	return func(ctx context.Context, err error) (int, any) {
		errcode := xerr.SERVER_COMMON_ERROR
		errmsg := xerr.ErrMsg(errcode)
		httpCode := http.StatusInternalServerError

		causeErr := errors.Cause(err)
		//zrpcErr.CodeMsg 是 RPC 错误（github.com/zeromicro/x/errors 里的类型）
		if e, ok := causeErr.(*zrpcErr.CodeMsg); ok && xerr.IsBusinessCode(e.Code) {
			errcode = e.Code
			errmsg = e.Msg
			// 业务错误通过响应体中的 code 表达，HTTP 层仍返回 200。
			httpCode = http.StatusOK
		} else {
			//status.FromError 来自 google.golang.org/grpc/status。如果错误是 gRPC status
			if gstatus, ok := status.FromError(causeErr); ok {
				// RPC 拦截器会把非业务错误转换成 codes.Internal。
				// 只有 xerr 中明确注册的自定义 code 才是业务错误。
				code := int(gstatus.Code())
				if xerr.IsBusinessCode(code) {
					errcode = code
					errmsg = gstatus.Message()
					httpCode = http.StatusOK
				}
			}
		}

		// 日志记录
		logx.WithContext(ctx).Errorf("【%s】 err %v", name, err)

		return httpCode, Fail(errcode, errmsg)
	}
}
