package job

import (
	"context"
	"time"

	"github.com/pkg/errors"
)

var ErrJobTimeout = errors.New("job timeout")

// RetryJetLagFunc 定义重试的时间策略
type RetryJetLagFunc func(ctx context.Context, retryCount int, lastTime time.Duration) time.Duration

// RetryJetLagAlways 默认固定间隔重试方法
func RetryJetLagAlways(ctx context.Context, retryCount int, lastTime time.Duration) time.Duration {
	return DefaultRetryJetLag
}

// IsRetryFunc 判断是否进行重试
type IsRetryFunc func(ctx context.Context, retryCount int, err error) bool

func RetryAlways(ctx context.Context, retryCount int, err error) bool {
	return true
}

func WithRetry(ctx context.Context, handler func(ctx context.Context) error, opts ...RetryOptions) error {
	opt := newOptions(opts...) // 合并默认值 + 传入选项

	// 判断程序本身是否设置了超时
	//如果调用方已经给 ctx 设了 deadline，尊重调用方，不覆盖
	_, ok := ctx.Deadline()
	if !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opt.timeout)
		defer cancel()
	}

	var (
		herr        error                 //保存最近一次 handler 的错误
		retryJetLag time.Duration         //保存上一次退避时长
		ch          = make(chan error, 1) //接收 handler 结果
	)

	//异步执行 + select 竞争
	for i := 0; i < opt.retryNums; i++ {
		// 启动一次尝试
		go func() {
			ch <- handler(ctx)
		}()
		//等待结果或超时
		select {
		case herr = <-ch: //ch 有数据 → handler 返回了，走结果处理
			if herr == nil {
				return nil // ① 成功，结束
			}
			if !opt.isRetryFunc(ctx, i, herr) {
				return herr // ② 不该重试，返回错误
			}
			retryJetLag = opt.retryJetLag(ctx, i, retryJetLag)
			time.Sleep(retryJetLag) // ③ 要重试，退避后下一轮
		case <-ctx.Done(): //ctx.Done()
			return ErrJobTimeout
		}
	}
	return herr
}
