package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/yourorg/auto-customer-service/internal/agent/adapter"
	"github.com/yourorg/auto-customer-service/internal/agent/adapter/wechat"
	"github.com/yourorg/auto-customer-service/internal/agent/idempotency"
	"github.com/yourorg/auto-customer-service/pkg/protocol"
)

func main() {
	// 初始化日志
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// 创建上下文，支持优雅退出
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 监听系统信号
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("收到退出信号，正在关闭...")
		cancel()
	}()

	// 初始化适配器
	adapterConfig := adapter.Config{
		EnableNative: true,
		EnableOCR:    true,
		EnableVisual: true,
		PollInterval: 1000,
		TimeoutMs:    5000,
	}

	wechatAdapter := wechat.NewWeChatAdapter()
	result := wechatAdapter.Init(adapterConfig)
	if result.Status != adapter.StatusSuccess {
		log.Fatalf("初始化适配器失败: %s", result.Error)
	}
	log.Println("适配器初始化成功")

	// 初始化幂等存储（TODO: 后续集成到 Agent 逻辑中）
	_ = idempotency.NewMemoryStore()
	log.Println("幂等存储初始化成功")

	// 初始化会话身份解析器（TODO: 后续集成到 Agent 逻辑中）
	_ = &protocol.DefaultIdentityResolver{}
	log.Println("会话身份解析器初始化成功")

	// TODO: 连接到 Gateway
	// err := client.Connect(ctx, "ws://localhost:8080/ws")
	// if err != nil {
	//     log.Fatalf("连接 Gateway 失败: %v", err)
	// }

	// 主循环
	log.Println("Agent 启动成功，等待任务...")
	<-ctx.Done()
	log.Println("Agent 已退出")
}
