package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/yourorg/auto-customer-service/internal/agent/comm"
	"github.com/yourorg/auto-customer-service/pkg/protocol"
)

// mockGatewayServer 模拟 Gateway WebSocket 服务器
type mockGatewayServer struct {
	server     *httptest.Server
	conn       *websocket.Conn
	mu         sync.Mutex
	messages   []protocol.Envelope
	clients    map[string]*websocket.Conn
	handler    func(env *protocol.Envelope)
}

// newMockGatewayServer 创建模拟 Gateway 服务器
func newMockGatewayServer() *mockGatewayServer {
	mgs := &mockGatewayServer{
		messages: make([]protocol.Envelope, 0),
		clients:  make(map[string]*websocket.Conn),
	}

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		// 获取查询参数
		sessionID := r.URL.Query().Get("session_id")
		_ = r.URL.Query().Get("device_id")   // deviceID is available but not used in this test
		_ = r.URL.Query().Get("tenant_id")   // tenantID is available but not used in this test

		mgs.mu.Lock()
		if sessionID != "" {
			mgs.clients[sessionID] = conn
		}
		mgs.conn = conn
		mgs.mu.Unlock()

		// 读取消息循环
		go func() {
			for {
				_, msg, err := conn.ReadMessage()
				if err != nil {
					return
				}

				var env protocol.Envelope
				if err := json.Unmarshal(msg, &env); err != nil {
					continue
				}

				mgs.mu.Lock()
				mgs.messages = append(mgs.messages, env)
				mgs.mu.Unlock()

				if mgs.handler != nil {
					mgs.handler(&env)
				}
			}
		}()
	})

	mgs.server = httptest.NewServer(handler)
	return mgs
}

// sendMessage 发送消息到客户端
func (mgs *mockGatewayServer) sendMessage(env *protocol.Envelope) error {
	mgs.mu.Lock()
	conn := mgs.conn
	mgs.mu.Unlock()

	if conn == nil {
		return nil
	}

	data, err := json.Marshal(env)
	if err != nil {
		return err
	}

	return conn.WriteMessage(websocket.TextMessage, data)
}

// getMessages 获取接收到的消息
func (mgs *mockGatewayServer) getMessages() []protocol.Envelope {
	mgs.mu.Lock()
	defer mgs.mu.Unlock()
	return append([]protocol.Envelope{}, mgs.messages...)
}

// close 关闭服务器
func (mgs *mockGatewayServer) close() {
	mgs.mu.Lock()
	defer mgs.mu.Unlock()

	if mgs.conn != nil {
		mgs.conn.Close()
	}
	for _, conn := range mgs.clients {
		conn.Close()
	}
	mgs.server.Close()
}

// TestGatewayAgent_E2E_Connection 测试 Gateway 与 Agent 的 WebSocket 连接
func TestGatewayAgent_E2E_Connection(t *testing.T) {
	// 创建模拟 Gateway 服务器
	gateway := newMockGatewayServer()
	defer gateway.close()

	// 提取服务器地址
	addr := strings.TrimPrefix(gateway.server.URL, "http://")

	// 创建 Agent WebSocket 客户端
	sessionID := protocol.GenerateSessionID()
	client := comm.NewWebSocketClient(sessionID, "device-001", "tenant-001")

	// 连接到 Gateway
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.Connect(ctx, addr)
	if err != nil {
		t.Fatalf("Failed to connect to gateway: %v", err)
	}
	defer client.Close()

	// 等待连接建立
	time.Sleep(100 * time.Millisecond)

	// 验证连接状态
	if !client.IsConnected() {
		t.Error("Client should be connected")
	}

	// 验证 sessionID
	if client.SessionID() != sessionID {
		t.Errorf("SessionID mismatch: got %v, want %v", client.SessionID(), sessionID)
	}
}

// TestGatewayAgent_E2E_SendCommand 测试 Agent 发送命令到 Gateway
func TestGatewayAgent_E2E_SendCommand(t *testing.T) {
	// 创建模拟 Gateway 服务器
	gateway := newMockGatewayServer()
	defer gateway.close()

	addr := strings.TrimPrefix(gateway.server.URL, "http://")

	// 创建 Agent WebSocket 客户端
	sessionID := protocol.GenerateSessionID()
	client := comm.NewWebSocketClient(sessionID, "device-001", "tenant-001")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.Connect(ctx, addr)
	if err != nil {
		t.Fatalf("Failed to connect to gateway: %v", err)
	}
	defer client.Close()

	time.Sleep(100 * time.Millisecond)

	// Agent 发送 conversation.new_message 事件
	newMessagePayload := protocol.NewMessagePayload{
		ConversationID: "conv_test_001",
		Message: protocol.MessageObs{
			MessageID:          "msg_test_001",
			ConversationID:     "conv_test_001",
			SenderSide:         "customer",
			NormalizedText:     "你好",
			Timestamp:          time.Now(),
			ObservedAt:         time.Now(),
			MessageFingerprint: "fp_test_001",
		},
	}

	eventEnvelope, err := protocol.NewEnvelope(
		protocol.KindEvent,
		protocol.PayloadNewMessage,
		newMessagePayload,
	)
	if err != nil {
		t.Fatalf("Failed to create event envelope: %v", err)
	}

	eventEnvelope.DeviceID = client.DeviceID()
	eventEnvelope.TenantID = client.TenantID()

	err = client.Send(eventEnvelope)
	if err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}

	// 等待消息到达服务器
	time.Sleep(200 * time.Millisecond)

	// 验证服务器接收到的消息
	messages := gateway.getMessages()
	if len(messages) == 0 {
		t.Error("Gateway should have received messages")
	}

	// 验证消息内容
	found := false
	for _, msg := range messages {
		if msg.PayloadType == protocol.PayloadNewMessage {
			found = true
			if msg.Kind != protocol.KindEvent {
				t.Errorf("Expected Kind Event, got %v", msg.Kind)
			}
			break
		}
	}

	if !found {
		t.Error("Gateway did not receive new_message event")
	}
}

// TestGatewayAgent_E2E_ReceiveCommand 测试 Agent 接收 Gateway 命令
func TestGatewayAgent_E2E_ReceiveCommand(t *testing.T) {
	// 创建模拟 Gateway 服务器
	gateway := newMockGatewayServer()
	defer gateway.close()

	addr := strings.TrimPrefix(gateway.server.URL, "http://")

	// 创建 Agent WebSocket 客户端
	sessionID := protocol.GenerateSessionID()
	client := comm.NewWebSocketClient(sessionID, "device-001", "tenant-001")

	// 注册命令处理器
	commandReceived := make(chan *protocol.Envelope, 1)
	client.RegisterHandler(protocol.PayloadReplyExecute, func(env *protocol.Envelope) {
		commandReceived <- env
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.Connect(ctx, addr)
	if err != nil {
		t.Fatalf("Failed to connect to gateway: %v", err)
	}
	defer client.Close()

	time.Sleep(100 * time.Millisecond)

	// Gateway 发送 reply.execute 命令
	replyPayload := protocol.ReplyExecutePayload{
		ConversationID: "conv_test_001",
		ReplyContent:   "您好！很高兴为您服务。",
	}

	commandEnvelope, err := protocol.NewEnvelope(
		protocol.KindCommand,
		protocol.PayloadReplyExecute,
		replyPayload,
	)
	if err != nil {
		t.Fatalf("Failed to create command envelope: %v", err)
	}

	commandEnvelope.TaskID = protocol.GenerateTaskID()

	err = gateway.sendMessage(commandEnvelope)
	if err != nil {
		t.Fatalf("Failed to send command: %v", err)
	}

	// 等待命令到达 Agent
	select {
	case env := <-commandReceived:
		if env.PayloadType != protocol.PayloadReplyExecute {
			t.Errorf("Expected PayloadType reply.execute, got %v", env.PayloadType)
		}
		if env.Kind != protocol.KindCommand {
			t.Errorf("Expected Kind Command, got %v", env.Kind)
		}

		// 解码载荷验证
		var payload protocol.ReplyExecutePayload
		err = protocol.DecodeEnvelopePayload(env, &payload)
		if err != nil {
			t.Fatalf("Failed to decode payload: %v", err)
		}

		if payload.ConversationID != "conv_test_001" {
			t.Errorf("ConversationID mismatch: got %v, want conv_test_001", payload.ConversationID)
		}
		if payload.ReplyContent != "您好！很高兴为您服务。" {
			t.Errorf("ReplyContent mismatch: got %v", payload.ReplyContent)
		}

	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for command")
	}
}

// TestGatewayAgent_E2E_TaskProgressFlow 测试任务进度流
func TestGatewayAgent_E2E_TaskProgressFlow(t *testing.T) {
	// 创建模拟 Gateway 服务器
	gateway := newMockGatewayServer()
	defer gateway.close()

	addr := strings.TrimPrefix(gateway.server.URL, "http://")

	// 创建 Agent WebSocket 客户端
	sessionID := protocol.GenerateSessionID()
	client := comm.NewWebSocketClient(sessionID, "device-001", "tenant-001")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.Connect(ctx, addr)
	if err != nil {
		t.Fatalf("Failed to connect to gateway: %v", err)
	}
	defer client.Close()

	time.Sleep(100 * time.Millisecond)

	taskID := protocol.GenerateTaskID()

	// 模拟发送多个进度事件
	progressStages := []struct {
		progress float64
		message  string
		stage    string
	}{
		{0.1, "检测应用实例中...", "detecting"},
		{0.3, "扫描会话列表中...", "scanning"},
		{0.5, "查找目标会话中...", "finding"},
		{0.7, "切换到目标会话...", "focusing"},
		{0.85, "发送消息中...", "sending"},
		{0.95, "验证消息发送中...", "verifying"},
	}

	for _, stage := range progressStages {
		payload := protocol.TaskProgressPayload{
			TaskID:   taskID,
			Progress: stage.progress,
			Message:  stage.message,
			Stage:    stage.stage,
		}

		env, err := protocol.NewEnvelope(
			protocol.KindEvent,
			protocol.PayloadTaskProgress,
			payload,
		)
		if err != nil {
			t.Fatalf("Failed to create progress envelope: %v", err)
		}

		env.DeviceID = client.DeviceID()
		env.TenantID = client.TenantID()
		env.TaskID = taskID

		err = client.Send(env)
		if err != nil {
			t.Fatalf("Failed to send progress event: %v", err)
		}

		time.Sleep(50 * time.Millisecond)
	}

	// 验证服务器接收到的进度事件
	time.Sleep(200 * time.Millisecond)
	messages := gateway.getMessages()

	progressCount := 0
	for _, msg := range messages {
		if msg.PayloadType == protocol.PayloadTaskProgress && msg.TaskID == taskID {
			progressCount++
		}
	}

	if progressCount != len(progressStages) {
		t.Errorf("Expected %d progress events, got %d", len(progressStages), progressCount)
	}
}

// TestGatewayAgent_E2E_TaskCompletedFlow 测试任务完成流
func TestGatewayAgent_E2E_TaskCompletedFlow(t *testing.T) {
	// 创建模拟 Gateway 服务器
	gateway := newMockGatewayServer()
	defer gateway.close()

	addr := strings.TrimPrefix(gateway.server.URL, "http://")

	// 创建 Agent WebSocket 客户端
	sessionID := protocol.GenerateSessionID()
	client := comm.NewWebSocketClient(sessionID, "device-001", "tenant-001")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.Connect(ctx, addr)
	if err != nil {
		t.Fatalf("Failed to connect to gateway: %v", err)
	}
	defer client.Close()

	time.Sleep(100 * time.Millisecond)

	taskID := protocol.GenerateTaskID()

	// 发送 task.completed 事件
	completedPayload := protocol.TaskCompletedPayload{
		TaskID:                     taskID,
		ObservedMessageFingerprint: "fp_verified",
		VerificationConfidence:     1.0,
		DeliveryState:              "verified",
	}

	env, err := protocol.NewEnvelope(
		protocol.KindEvent,
		protocol.PayloadTaskCompleted,
		completedPayload,
	)
	if err != nil {
		t.Fatalf("Failed to create task.completed envelope: %v", err)
	}

	env.DeviceID = client.DeviceID()
	env.TenantID = client.TenantID()
	env.TaskID = taskID

	err = client.Send(env)
	if err != nil {
		t.Fatalf("Failed to send task.completed event: %v", err)
	}

	// 验证服务器接收到的消息
	time.Sleep(200 * time.Millisecond)
	messages := gateway.getMessages()

	found := false
	for _, msg := range messages {
		if msg.PayloadType == protocol.PayloadTaskCompleted && msg.TaskID == taskID {
			found = true

			// 解码载荷验证
			var payload protocol.TaskCompletedPayload
			err = protocol.DecodeEnvelopePayload(&msg, &payload)
			if err != nil {
				t.Fatalf("Failed to decode payload: %v", err)
			}

			if payload.ObservedMessageFingerprint != "fp_verified" {
				t.Errorf("Fingerprint mismatch: got %v, want fp_verified", payload.ObservedMessageFingerprint)
			}
			if payload.VerificationConfidence != 1.0 {
				t.Errorf("Confidence mismatch: got %v, want 1.0", payload.VerificationConfidence)
			}
			break
		}
	}

	if !found {
		t.Error("Gateway did not receive task.completed event")
	}
}

// TestGatewayAgent_E2E_TaskFailedFlow 测试任务失败流
func TestGatewayAgent_E2E_TaskFailedFlow(t *testing.T) {
	// 创建模拟 Gateway 服务器
	gateway := newMockGatewayServer()
	defer gateway.close()

	addr := strings.TrimPrefix(gateway.server.URL, "http://")

	// 创建 Agent WebSocket 客户端
	sessionID := protocol.GenerateSessionID()
	client := comm.NewWebSocketClient(sessionID, "device-001", "tenant-001")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.Connect(ctx, addr)
	if err != nil {
		t.Fatalf("Failed to connect to gateway: %v", err)
	}
	defer client.Close()

	time.Sleep(100 * time.Millisecond)

	taskID := protocol.GenerateTaskID()

	// 发送 task.failed 事件
	failedPayload := protocol.TaskFailedPayload{
		TaskID:      taskID,
		ErrorCode:   "SEND_FAILED",
		ErrorReason: "Failed to send message",
	}

	env, err := protocol.NewEnvelope(
		protocol.KindEvent,
		protocol.PayloadTaskFailed,
		failedPayload,
	)
	if err != nil {
		t.Fatalf("Failed to create task.failed envelope: %v", err)
	}

	env.DeviceID = client.DeviceID()
	env.TenantID = client.TenantID()
	env.TaskID = taskID

	err = client.Send(env)
	if err != nil {
		t.Fatalf("Failed to send task.failed event: %v", err)
	}

	// 验证服务器接收到的消息
	time.Sleep(200 * time.Millisecond)
	messages := gateway.getMessages()

	found := false
	for _, msg := range messages {
		if msg.PayloadType == protocol.PayloadTaskFailed && msg.TaskID == taskID {
			found = true

			// 解码载荷验证
			var payload protocol.TaskFailedPayload
			err = protocol.DecodeEnvelopePayload(&msg, &payload)
			if err != nil {
				t.Fatalf("Failed to decode payload: %v", err)
			}

			if payload.ErrorCode != "SEND_FAILED" {
				t.Errorf("ErrorCode mismatch: got %v, want SEND_FAILED", payload.ErrorCode)
			}
			if payload.ErrorReason != "Failed to send message" {
				t.Errorf("ErrorReason mismatch: got %v", payload.ErrorReason)
			}
			break
		}
	}

	if !found {
		t.Error("Gateway did not receive task.failed event")
	}
}

// TestGatewayAgent_E2E_MultipleCommands 测试接收多个命令
func TestGatewayAgent_E2E_MultipleCommands(t *testing.T) {
	// 创建模拟 Gateway 服务器
	gateway := newMockGatewayServer()
	defer gateway.close()

	addr := strings.TrimPrefix(gateway.server.URL, "http://")

	// 创建 Agent WebSocket 客户端
	sessionID := protocol.GenerateSessionID()
	client := comm.NewWebSocketClient(sessionID, "device-001", "tenant-001")

	// 注册多个命令处理器
	replyExecuteCount := 0
	convModeSetCount := 0
	diagnosticCaptureCount := 0

	client.RegisterHandler(protocol.PayloadReplyExecute, func(env *protocol.Envelope) {
		replyExecuteCount++
	})

	client.RegisterHandler(protocol.PayloadConvModeSet, func(env *protocol.Envelope) {
		convModeSetCount++
	})

	client.RegisterHandler(protocol.PayloadDiagnosticCapture, func(env *protocol.Envelope) {
		diagnosticCaptureCount++
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.Connect(ctx, addr)
	if err != nil {
		t.Fatalf("Failed to connect to gateway: %v", err)
	}
	defer client.Close()

	time.Sleep(100 * time.Millisecond)

	// 发送多个不同类型的命令
	commands := []struct {
		payloadType protocol.PayloadType
		payload     protocol.Payload
	}{
		{
			protocol.PayloadReplyExecute,
			protocol.ReplyExecutePayload{
				ConversationID: "conv_001",
				ReplyContent:   "Test message",
			},
		},
		{
			protocol.PayloadConvModeSet,
			protocol.ConvModeSetPayload{
				ConversationID: "conv_001",
				Mode:           "auto",
			},
		},
		{
			protocol.PayloadDiagnosticCapture,
			protocol.DiagnosticCapturePayload{
				CaptureType:    "full",
				ConversationID: "conv_001",
			},
		},
	}

	for _, cmd := range commands {
		env, err := protocol.NewEnvelope(
			protocol.KindCommand,
			cmd.payloadType,
			cmd.payload,
		)
		if err != nil {
			t.Fatalf("Failed to create command envelope: %v", err)
		}

		env.TaskID = protocol.GenerateTaskID()

		err = gateway.sendMessage(env)
		if err != nil {
			t.Fatalf("Failed to send command: %v", err)
		}

		time.Sleep(50 * time.Millisecond)
	}

	// 等待所有命令处理完成
	time.Sleep(300 * time.Millisecond)

	// 验证所有处理器都被调用
	if replyExecuteCount != 1 {
		t.Errorf("Expected 1 reply.execute call, got %d", replyExecuteCount)
	}
	if convModeSetCount != 1 {
		t.Errorf("Expected 1 conversation.mode.set call, got %d", convModeSetCount)
	}
	if diagnosticCaptureCount != 1 {
		t.Errorf("Expected 1 diagnostic.capture call, got %d", diagnosticCaptureCount)
	}
}
