// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package main

import (
	"chat-go/pkg/configserver"
	"chat-go/pkg/resultx"
	"flag"
	"fmt"
	"sync"

	"chat-go/apps/user/api/internal/config"
	"chat-go/apps/user/api/internal/handler"
	"chat-go/apps/user/api/internal/svc"

	"github.com/zeromicro/go-zero/core/proc"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
)

var configFile = flag.String("f", "etc/user.yaml", "the config file")

// 🔥 并发同步管道：启动 N 个 goroutine，等它们全部完成之后再继续下一步
// 让 main 阻塞等待所有 Run() 安全退出

/*
配置变更回调
→ 解析新配置到局部 c
→ proc.WrapUp()        通知当前服务优雅停止
→ go Run(c)            用新配置启动新服务实例
→ wg 等待所有实例退出
*/
var wg sync.WaitGroup

func main() {
	flag.Parse()

	var c config.Config

	// go-zero配置默认加载方式
	//conf.MustLoad(*configFile, &c)

	var configs = "user-api.yaml"

	// 配置中心加载方式
	err := configserver.NewConfigServer(*configFile, configserver.NewSail(&configserver.Config{
		ETCDEndpoints:  "127.0.0.1:3379",
		ProjectKey:     "3c46a0407be60a1f00731ab8e9575df2",
		Namespace:      "user",
		Configs:        configs,
		ConfigFilePath: "../etc/conf",
		LogLevel:       "DEBUG",
	})).MustLoad(&c, func(bytes []byte) error { // 回调函数（配置更新后的处理）
		//只是打印一下
		var c config.Config
		err := configserver.LoadFromJsonBytes(bytes, &c)
		if err != nil {
			fmt.Println("config read err :", err)
		}
		fmt.Printf(configs, "config has changed : %+v\n", c)

		/*
		 * 1.平滑重启可以从 API 和 RPC 两个角度来讲，API的平滑重启涉及监听服务停止信号并优雅地关闭程序。
		 * 2.在API的执行流程中，通过监听信号变量来判断服务是否停止，并执行优雅关闭程序的方法。
		 * 3.Go标准库中的signal包提供了处理频繁重启的方法，包括发送信号和通知服务停止。
		 * ✨ go-zero -> core -> proc.WrapUp():
		 *  - 通知当前运行中的服务 server.Start() 内部退出循环
		 *  - 触发 defer server.Stop()，开始优雅关闭（停止接收新请求，清理资源）
		 */
		proc.WrapUp() // 通知当前服务优雅停止

		// 启动新的服务实例（使用新配置），并加入 WaitGroup
		wg.Add(1) // 阻塞
		go func(c config.Config) {
			defer wg.Done()
			Run(c) // 新服务开始运行，阻塞于 server.Start()
		}(c)

		return nil
	})
	if err != nil {
		panic(err)
	}

	// 程序启动后第一次运行 API 服务（使用初始配置）
	wg.Add(1)
	go func(c config.Config) {
		defer wg.Done()
		Run(c)
	}(c)

	//✨ main goroutine 等待所有服务（当前和未来因配置更新启动的服务）优雅退出
	wg.Wait()
}

// Run 启动一个 go-zero 的 REST API 服务，server.Start() 内部阻塞运行
func Run(c config.Config) {
	// 创建 REST API 服务，启用 CORS（跨域支持）
	server := rest.MustNewServer(c.RestConf, rest.WithCors())
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	httpx.SetErrorHandlerCtx(resultx.ErrHandler(c.Name))
	httpx.SetOkHandler(resultx.OkHandler)

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
