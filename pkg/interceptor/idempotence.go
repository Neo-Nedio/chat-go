package interceptor

import (
	"chat-go/pkg/xerr"
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/utils"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

type Idempotent interface {
	// Identify 生成/获取这次请求的唯一标识（幂等 key）
	Identify(ctx context.Context, method string) string
	// IsIdempotentMethod 判断这个方法是否需要幂等处理（不是所有方法都要）
	IsIdempotentMethod(fullMethod string) bool
	// TryAcquire 幂等性的验证
	TryAcquire(ctx context.Context, id string) (resp any, isAcquire bool)
	// SaveResp 执行完保存结果，供后续重复请求直接返回
	SaveResp(ctx context.Context, id string, resp any, respErr error) error
}

var (
	// TKey 请求任务标识
	TKey = "chat-idempotence-task-id"
	// DKey 设置rpc调度中的rpc请求的标识
	DKey = "chat-idempotence-dispatch-key"
)

// ContextWithVal 添加到上下文方便客户端获取
// 给 context 里塞一个唯一请求 ID（UUID），方便整条调用链路上取到这个 ID，用于日志追踪、链路串联
func ContextWithVal(ctx context.Context) context.Context {
	// 设置请求id
	return context.WithValue(ctx, TKey, utils.NewUuid())
}

// NewIdempotenceClient grpc客户端的拦截器
// 在每次 gRPC 调用发出前，给请求打上一个“幂等标识”，塞进 metadata，让服务端能识别这是不是同一个请求
func NewIdempotenceClient(idempotent Idempotent) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// 获取唯一的key
		identify := idempotent.Identify(ctx, method)
		// 在rpc请求中设置头部信息
		ctx = metadata.NewOutgoingContext(ctx, map[string][]string{
			DKey: []string{identify},
		})
		// 请求
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// NewIdempotenceServer grpc服务端的拦截器
func NewIdempotenceServer(idempotent Idempotent) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		// 获取请求id
		identify := metadata.ValueFromIncomingContext(ctx, DKey)
		if len(identify) == 0 || !idempotent.IsIdempotentMethod(info.FullMethod) {
			// 没 key 或不需要幂等，直接执行
			return handler(ctx, req)
		}

		fmt.Println("----", "请求进行幂等处理", identify)
		//isAcquire = true：抢到了执行权，说明这个 key 之前没执行过/没在执行。
		//isAcquire = false：没抢到，说明 key 已存在（正在执行或已执行完），r 是上次的结果（可能为 nil）
		r, isAcquire := idempotent.TryAcquire(ctx, identify[0])
		if isAcquire {
			resp, err = handler(ctx, req)
			fmt.Println("---- 执行任务")
			// 保存执行之后的结果
			if err := idempotent.SaveResp(ctx, identify[0], resp, err); err != nil {
				return resp, err
			}
			return resp, err
		}
		// 任务已经执行完了
		if r != nil {
			fmt.Println("----", "任务已经执行完了")
			return r, nil
		}
		// 任务还在执行
		//🔥注意：因为需要grpc的重试，所以此处需要使用grpc的错误码
		return nil, errors.WithStack(xerr.New(int(codes.DeadlineExceeded), fmt.Sprintf("存在其他任务在执行"+
			"id %v", identify[0])))
	}
}
