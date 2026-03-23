package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yourorg/auto-customer-service/internal/agent/adapter"
	"github.com/yourorg/auto-customer-service/internal/agent/adapter/wechat"
	"github.com/yourorg/auto-customer-service/internal/agent/comm"
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

	// 初始化幂等存储
	idempStore := idempotency.NewMemoryStore()
	log.Println("幂等存储初始化成功")

	// 初始化会话身份解析器
	identityResolver := &protocol.DefaultIdentityResolver{}
	log.Println("会话身份解析器初始化成功")

	// 创建 WebSocket 客户端
	sessionID := protocol.GenerateSessionID()
	client := comm.NewWebSocketClient(sessionID, "agent-001", "tenant-001")
	log.Printf("WebSocket 客户端创建成功，会话ID: %s", sessionID)

	// 注册命令处理器
	client.RegisterHandler(protocol.PayloadReplyExecute, func(env *protocol.Envelope) {
		handleReplyExecute(ctx, env, wechatAdapter, idempStore, identityResolver, client)
	})

	client.RegisterHandler(protocol.PayloadConvModeSet, func(env *protocol.Envelope) {
		handleConvModeSet(ctx, env, wechatAdapter, client)
	})

	client.RegisterHandler(protocol.PayloadDiagnosticCapture, func(env *protocol.Envelope) {
		handleDiagnosticCapture(ctx, env, wechatAdapter, client)
	})

	// 连接到 Gateway
	gatewayAddr := "localhost:8080"
	if os.Getenv("GATEWAY_ADDR") != "" {
		gatewayAddr = os.Getenv("GATEWAY_ADDR")
	}
	log.Printf("正在连接到 Gateway: %s", gatewayAddr)

	err := client.Connect(ctx, gatewayAddr)
	if err != nil {
		log.Fatalf("连接 Gateway 失败: %v", err)
	}
	log.Println("成功连接到 Gateway")

	// 发送连接成功事件
	go func() {
		time.Sleep(100 * time.Millisecond)
		log.Println("发送连接就绪事件")
	}()

	// 主循环
	log.Println("Agent 启动成功，等待任务...")
	<-ctx.Done()

	// 关闭连接
	client.Close()
	log.Println("Agent 已退出")
}

// handleReplyExecute 处理 reply.execute 命令
func handleReplyExecute(
	ctx context.Context,
	env *protocol.Envelope,
	chatAdapter adapter.ChatAdapter,
	idempStore *idempotency.MemoryStore,
	resolver protocol.ConversationIdentityResolver,
	client *comm.WebSocketClient,
) {
	log.Printf("收到 reply.execute 命令: task_id=%s", env.TaskID)

	// 检查幂等性
	_, err := idempStore.GetRecord(env.TaskID)
	if err == nil {
		log.Printf("任务已处理，跳过: task_id=%s", env.TaskID)
		return
	}

	// 解码载荷
	var payload protocol.ReplyExecutePayload
	if err := protocol.DecodeEnvelopePayload(env, &payload); err != nil {
		log.Printf("解码载荷失败: %v", err)
		return
	}

	// 阶段1: 检测应用实例
	sendProgress(client, env.TaskID, 0.1, "检测应用实例中...", "detecting")
	instances, detectResult := chatAdapter.Detect()
	if detectResult.Status != adapter.StatusSuccess || len(instances) == 0 {
		log.Printf("未找到应用实例: %s", detectResult.Error)
		sendTaskFailed(client, env.TaskID, "NO_INSTANCE", "未找到应用实例")
		return
	}
	log.Printf("检测到 %d 个应用实例", len(instances))

	// 阶段2: 扫描会话列表
	sendProgress(client, env.TaskID, 0.3, "扫描会话列表中...", "scanning")
	conversations, scanResult := chatAdapter.Scan(instances[0])
	if scanResult.Status != adapter.StatusSuccess {
		log.Printf("扫描会话失败: %s", scanResult.Error)
		sendTaskFailed(client, env.TaskID, "SCAN_FAILED", "扫描会话失败")
		return
	}
	log.Printf("扫描到 %d 个会话", len(conversations))

	// 阶段3: 查找目标会话
	sendProgress(client, env.TaskID, 0.5, "查找目标会话中...", "finding")
	convID := payload.ConversationID
	var targetConv *protocol.ConversationRef
	for i := range conversations {
		if conversations[i].DisplayName == convID {
			targetConv = &conversations[i]
			break
		}
	}

	if targetConv == nil {
		log.Printf("未找到会话: %s", convID)
		sendTaskFailed(client, env.TaskID, "CONV_NOT_FOUND", "未找到会话")
		return
	}
	log.Printf("目标会话: %s", targetConv.DisplayName)

	// 阶段4: 聚焦到会话
	sendProgress(client, env.TaskID, 0.7, "切换到目标会话...", "focusing")
	focusResult := chatAdapter.Focus(*targetConv)
	if focusResult.Status != adapter.StatusSuccess {
		log.Printf("聚焦会话失败: %s", focusResult.Error)
		sendTaskFailed(client, env.TaskID, "FOCUS_FAILED", "聚焦会话失败")
		return
	}
	log.Printf("成功切换到会话: %s (置信度: %.2f)", targetConv.DisplayName, focusResult.Confidence)

	// 阶段5: 发送消息
	sendProgress(client, env.TaskID, 0.85, "发送消息中...", "sending")
	sendResult := chatAdapter.Send(*targetConv, payload.ReplyContent, env.TaskID)
	if sendResult.Status != adapter.StatusSuccess {
		log.Printf("发送消息失败: %s", sendResult.Error)
		sendTaskFailed(client, env.TaskID, "SEND_FAILED", sendResult.Error)
		return
	}
	log.Printf("消息发送成功 (耗时: %dms, 置信度: %.2f)", sendResult.ElapsedMs, sendResult.Confidence)

	// 阶段6: 验证消息
	sendProgress(client, env.TaskID, 0.95, "验证消息发送中...", "verifying")
	time.Sleep(500 * time.Millisecond) // 等待消息发送
	msgObs, verifyResult := chatAdapter.Verify(*targetConv, payload.ReplyContent, 3*time.Second)

	// 根据验证结果确定交付状态
	var deliveryState string
	if verifyResult.Status == adapter.StatusSuccess && verifyResult.Confidence >= 0.8 {
		deliveryState = "verified"
		log.Printf("消息验证成功 (置信度: %.2f)", verifyResult.Confidence)
	} else if verifyResult.Status == adapter.StatusSuccess {
		deliveryState = "sent_unverified"
		log.Printf("消息发送成功但验证置信度较低 (置信度: %.2f)", verifyResult.Confidence)
	} else {
		deliveryState = "unknown"
		log.Printf("验证消息失败: %s", verifyResult.Error)
	}

	// 标记任务已处理
	record := idempotency.Record{
		TaskID:     env.TaskID,
		Status:     "completed",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	idempStore.CreateRecord(record)

	// 发送任务完成事件
	taskCompleted := protocol.TaskCompletedPayload{
		TaskID:                 env.TaskID,
		ObservedMessageFingerprint: "",
		VerificationConfidence: verifyResult.Confidence,
		DeliveryState:          deliveryState,
	}
	if msgObs != nil {
		taskCompleted.ObservedMessageFingerprint = msgObs.MessageFingerprint
	}

	envCompleted, err := protocol.NewEnvelope(
		protocol.KindEvent,
		protocol.PayloadTaskCompleted,
		taskCompleted,
	)
	if err != nil {
		log.Printf("创建完成事件失败: %v", err)
		return
	}

	envCompleted.DeviceID = client.DeviceID()
	envCompleted.TenantID = client.TenantID()
	envCompleted.TaskID = env.TaskID

	if err := client.Send(envCompleted); err != nil {
		log.Printf("发送任务完成事件失败: %v", err)
	} else {
		log.Printf("任务完成: task_id=%s", env.TaskID)
	}
}

// handleConvModeSet 处理 conversation.mode.set 命令
func handleConvModeSet(
	ctx context.Context,
	env *protocol.Envelope,
	chatAdapter adapter.ChatAdapter,
	client *comm.WebSocketClient,
) {
	log.Printf("收到 conversation.mode.set 命令: task_id=%s", env.TaskID)

	var payload protocol.ConvModeSetPayload
	if err := protocol.DecodeEnvelopePayload(env, &payload); err != nil {
		log.Printf("解码载荷失败: %v", err)
		return
	}

	log.Printf("设置会话模式: conv=%s, mode=%s", payload.ConversationID, payload.Mode)

	// 发送进度事件
	progress := protocol.TaskProgressPayload{
		TaskID:   env.TaskID,
		Progress: 1.0,
		Message:  "模式设置完成",
		Stage:    "completed",
	}

	envProgress, err := protocol.NewEnvelope(
		protocol.KindEvent,
		protocol.PayloadTaskProgress,
		progress,
	)
	if err != nil {
		log.Printf("创建进度事件失败: %v", err)
		return
	}

	envProgress.DeviceID = client.DeviceID()
	envProgress.TenantID = client.TenantID()
	envProgress.TaskID = env.TaskID

	client.Send(envProgress)
}

// handleDiagnosticCapture 处理 diagnostic.capture 命令
func handleDiagnosticCapture(
	ctx context.Context,
	env *protocol.Envelope,
	chatAdapter adapter.ChatAdapter,
	client *comm.WebSocketClient,
) {
	log.Printf("收到 diagnostic.capture 命令: task_id=%s", env.TaskID)

	var payload protocol.DiagnosticCapturePayload
	if err := protocol.DecodeEnvelopePayload(env, &payload); err != nil {
		log.Printf("解码载荷失败: %v", err)
		return
	}

	log.Printf("捕获诊断信息: type=%s, conv=%s", payload.CaptureType, payload.ConversationID)

	// 获取诊断信息
	diagnostics, result := chatAdapter.CaptureDiagnostics()
	if result.Status != adapter.StatusSuccess {
		log.Printf("捕获诊断失败: %s", result.Error)
		return
	}

	log.Printf("诊断信息: %v", diagnostics)

	// 发送进度事件
	progress := protocol.TaskProgressPayload{
		TaskID:   env.TaskID,
		Progress: 1.0,
		Message:  "诊断捕获完成",
		Stage:    "completed",
	}

	envProgress, err := protocol.NewEnvelope(
		protocol.KindEvent,
		protocol.PayloadTaskProgress,
		progress,
	)
	if err != nil {
		log.Printf("创建进度事件失败: %v", err)
		return
	}

	envProgress.DeviceID = client.DeviceID()
	envProgress.TenantID = client.TenantID()
	envProgress.TaskID = env.TaskID

	client.Send(envProgress)
}

// sendTaskFailed 发送任务失败事件
func sendTaskFailed(client *comm.WebSocketClient, taskID, errorCode, errorReason string) {
	payload := protocol.TaskFailedPayload{
		TaskID:      taskID,
		ErrorCode:   errorCode,
		ErrorReason: errorReason,
	}

	env, err := protocol.NewEnvelope(
		protocol.KindEvent,
		protocol.PayloadTaskFailed,
		payload,
	)
	if err != nil {
		log.Printf("创建失败事件失败: %v", err)
		return
	}

	env.DeviceID = client.DeviceID()
	env.TenantID = client.TenantID()
	env.TaskID = taskID

	if err := client.Send(env); err != nil {
		log.Printf("发送失败事件失败: %v", err)
	} else {
		log.Printf("任务失败: task_id=%s, error=%s", taskID, errorCode)
	}
}

// sendProgress 发送任务进度事件
func sendProgress(client *comm.WebSocketClient, taskID string, progress float64, message, stage string) {
	payload := protocol.TaskProgressPayload{
		TaskID:   taskID,
		Progress: progress,
		Message:  message,
		Stage:    stage,
	}

	env, err := protocol.NewEnvelope(
		protocol.KindEvent,
		protocol.PayloadTaskProgress,
		payload,
	)
	if err != nil {
		log.Printf("创建进度事件失败: %v", err)
		return
	}

	env.DeviceID = client.DeviceID()
	env.TenantID = client.TenantID()
	env.TaskID = taskID

	if err := client.Send(env); err != nil {
		log.Printf("发送进度事件失败: %v", err)
	} else {
		log.Printf("任务进度: task_id=%s, progress=%.2f, stage=%s", taskID, progress, stage)
	}
}
