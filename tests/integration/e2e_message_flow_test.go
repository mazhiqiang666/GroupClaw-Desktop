package integration

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/yourorg/auto-customer-service/internal/agent/idempotency"
	"github.com/yourorg/auto-customer-service/pkg/protocol"
)

// TestE2E_NewMessageToReplyExecute 测试完整消息流：new_message -> reply.execute -> send -> verify -> task.completed
func TestE2E_NewMessageToReplyExecute(t *testing.T) {
	// 1. Agent 上报 conversation.new_message 事件
	newMessagePayload := protocol.NewMessagePayload{
		ConversationID: "conv_test_001",
		Message: protocol.MessageObs{
			MessageID:          "msg_test_001",
			ConversationID:     "conv_test_001",
			SenderSide:         "customer",
			NormalizedText:     "你好，请问有什么可以帮助您的？",
			Timestamp:          time.Now(),
			ObservedAt:         time.Now(),
			MessageFingerprint: "fp_test_001",
		},
	}

	// 创建事件信封
	eventEnvelope, err := protocol.NewEnvelope(
		protocol.KindEvent,
		protocol.PayloadNewMessage,
		newMessagePayload,
	)
	if err != nil {
		t.Fatalf("Failed to create event envelope: %v", err)
	}

	// 验证事件信封
	if eventEnvelope.Kind != protocol.KindEvent {
		t.Errorf("Expected Kind Event, got %v", eventEnvelope.Kind)
	}
	if eventEnvelope.PayloadType != protocol.PayloadNewMessage {
		t.Errorf("Expected PayloadType conversation.new_message, got %v", eventEnvelope.PayloadType)
	}

	// 2. Gateway 下发 reply.execute 命令
	replyPayload := protocol.ReplyExecutePayload{
		ConversationID: "conv_test_001",
		ReplyContent:   "您好！很高兴为您服务。",
	}

	// 创建命令信封
	commandEnvelope, err := protocol.NewEnvelope(
		protocol.KindCommand,
		protocol.PayloadReplyExecute,
		replyPayload,
	)
	if err != nil {
		t.Fatalf("Failed to create command envelope: %v", err)
	}

	// 验证命令信封
	if commandEnvelope.Kind != protocol.KindCommand {
		t.Errorf("Expected Kind Command, got %v", commandEnvelope.Kind)
	}
	if commandEnvelope.PayloadType != protocol.PayloadReplyExecute {
		t.Errorf("Expected PayloadType reply.execute, got %v", commandEnvelope.PayloadType)
	}

	// 3. Agent 执行 send + verify 工作流
	store := idempotency.NewMemoryStore()
	mockAdapter := NewMockAdapter()
	runner := NewWorkflowRunner(store, mockAdapter)

	taskID := protocol.GenerateTaskID()
	dedupeKey := "e2e_test_001"
	conversationID := "conv_test_001"
	content := "您好！很高兴为您服务。"

	// 执行工作流
	err = runner.RunSendVerifyWorkflow(taskID, dedupeKey, conversationID, content)
	if err != nil {
		t.Fatalf("Workflow failed: %v", err)
	}

	// 4. Agent 回传 task.progress / task.completed 事件
	// 验证最终状态
	retrieved, err := store.GetRecord(taskID)
	if err != nil {
		t.Fatalf("GetRecord failed: %v", err)
	}

	if retrieved.Status != "verified" {
		t.Errorf("Final status should be 'verified', got %v", retrieved.Status)
	}

	// 验证 task.completed 事件载荷
	taskCompletedPayload := protocol.TaskCompletedPayload{
		TaskID:                     taskID,
		ObservedMessageFingerprint: "fp_verified",
		VerificationConfidence:     1.0,
		DeliveryState:              "verified",
	}

	// 创建 task.completed 事件信封
	taskCompletedEnvelope, err := protocol.NewEnvelope(
		protocol.KindEvent,
		protocol.PayloadTaskCompleted,
		taskCompletedPayload,
	)
	if err != nil {
		t.Fatalf("Failed to create task.completed envelope: %v", err)
	}

	// 验证 task.completed 信封
	if taskCompletedEnvelope.Kind != protocol.KindEvent {
		t.Errorf("Expected Kind Event for task.completed, got %v", taskCompletedEnvelope.Kind)
	}
	if taskCompletedEnvelope.PayloadType != protocol.PayloadTaskCompleted {
		t.Errorf("Expected PayloadType task.completed, got %v", taskCompletedEnvelope.PayloadType)
	}

	// 验证 MockAdapter 被调用
	sendCalls := mockAdapter.GetSendCalls()
	if len(sendCalls) != 1 {
		t.Errorf("Expected 1 Send call, got %d", len(sendCalls))
	}

	verifyCalls := mockAdapter.GetVerifyCalls()
	if len(verifyCalls) != 1 {
		t.Errorf("Expected 1 Verify call, got %d", len(verifyCalls))
	}
}

// TestE2E_TaskProgressEvent 测试 task.progress 事件
func TestE2E_TaskProgressEvent(t *testing.T) {
	taskID := protocol.GenerateTaskID()

	progressPayload := protocol.TaskProgressPayload{
		TaskID:   taskID,
		Progress: 0.5,
		Message:  "发送中...",
		Stage:    "sending",
	}

	// 创建 task.progress 事件信封
	envelope, err := protocol.NewEnvelope(
		protocol.KindEvent,
		protocol.PayloadTaskProgress,
		progressPayload,
	)
	if err != nil {
		t.Fatalf("Failed to create task.progress envelope: %v", err)
	}

	// 验证信封
	if envelope.Kind != protocol.KindEvent {
		t.Errorf("Expected Kind Event, got %v", envelope.Kind)
	}
	if envelope.PayloadType != protocol.PayloadTaskProgress {
		t.Errorf("Expected PayloadType task.progress, got %v", envelope.PayloadType)
	}

	// 解码载荷验证
	var decoded protocol.TaskProgressPayload
	err = protocol.DecodeEnvelopePayload(envelope, &decoded)
	if err != nil {
		t.Fatalf("Failed to decode task.progress payload: %v", err)
	}

	if decoded.TaskID != taskID {
		t.Errorf("TaskID mismatch: got %v, want %v", decoded.TaskID, taskID)
	}
	if decoded.Progress != 0.5 {
		t.Errorf("Progress mismatch: got %v, want 0.5", decoded.Progress)
	}
	if decoded.Stage != "sending" {
		t.Errorf("Stage mismatch: got %v, want 'sending'", decoded.Stage)
	}
}

// TestE2E_TaskFailedEvent 测试 task.failed 事件
func TestE2E_TaskFailedEvent(t *testing.T) {
	taskID := protocol.GenerateTaskID()

	failedPayload := protocol.TaskFailedPayload{
		TaskID:      taskID,
		ErrorCode:   "SEND_FAILED",
		ErrorReason: "Failed to send message",
	}

	// 创建 task.failed 事件信封
	envelope, err := protocol.NewEnvelope(
		protocol.KindEvent,
		protocol.PayloadTaskFailed,
		failedPayload,
	)
	if err != nil {
		t.Fatalf("Failed to create task.failed envelope: %v", err)
	}

	// 验证信封
	if envelope.Kind != protocol.KindEvent {
		t.Errorf("Expected Kind Event, got %v", envelope.Kind)
	}
	if envelope.PayloadType != protocol.PayloadTaskFailed {
		t.Errorf("Expected PayloadType task.failed, got %v", envelope.PayloadType)
	}

	// 解码载荷验证
	var decoded protocol.TaskFailedPayload
	err = protocol.DecodeEnvelopePayload(envelope, &decoded)
	if err != nil {
		t.Fatalf("Failed to decode task.failed payload: %v", err)
	}

	if decoded.TaskID != taskID {
		t.Errorf("TaskID mismatch: got %v, want %v", decoded.TaskID, taskID)
	}
	if decoded.ErrorCode != "SEND_FAILED" {
		t.Errorf("ErrorCode mismatch: got %v, want 'SEND_FAILED'", decoded.ErrorCode)
	}
}

// TestE2E_FullRoundTrip 测试完整序列化/反序列化轮转
func TestE2E_FullRoundTrip(t *testing.T) {
	// 1. 创建原始数据
	taskID := protocol.GenerateTaskID()
	originalPayload := protocol.TaskCompletedPayload{
		TaskID:                     taskID,
		ObservedMessageFingerprint: "fp_test_123",
		VerificationConfidence:     0.95,
		DeliveryState:              "delivered",
	}

	// 2. 创建信封
	envelope, err := protocol.NewEnvelope(
		protocol.KindEvent,
		protocol.PayloadTaskCompleted,
		originalPayload,
	)
	if err != nil {
		t.Fatalf("Failed to create envelope: %v", err)
	}

	// 3. 序列化信封
	data, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("Marshal envelope failed: %v", err)
	}

	// 4. 反序列化信封
	var decodedEnvelope protocol.Envelope
	err = json.Unmarshal(data, &decodedEnvelope)
	if err != nil {
		t.Fatalf("Unmarshal envelope failed: %v", err)
	}

	// 5. 解码载荷
	var decodedPayload protocol.TaskCompletedPayload
	err = protocol.DecodeEnvelopePayload(&decodedEnvelope, &decodedPayload)
	if err != nil {
		t.Fatalf("DecodeEnvelopePayload failed: %v", err)
	}

	// 6. 验证完整轮转
	if decodedPayload.TaskID != originalPayload.TaskID {
		t.Errorf("TaskID mismatch: got %v, want %v", decodedPayload.TaskID, originalPayload.TaskID)
	}
	if decodedPayload.ObservedMessageFingerprint != originalPayload.ObservedMessageFingerprint {
		t.Errorf("ObservedMessageFingerprint mismatch: got %v, want %v",
			decodedPayload.ObservedMessageFingerprint, originalPayload.ObservedMessageFingerprint)
	}
	if decodedPayload.VerificationConfidence != originalPayload.VerificationConfidence {
		t.Errorf("VerificationConfidence mismatch: got %v, want %v",
			decodedPayload.VerificationConfidence, originalPayload.VerificationConfidence)
	}
	if decodedPayload.DeliveryState != originalPayload.DeliveryState {
		t.Errorf("DeliveryState mismatch: got %v, want %v",
			decodedPayload.DeliveryState, originalPayload.DeliveryState)
	}
}

// TestE2E_SendVerifyFailure 测试发送验证失败场景
func TestE2E_SendVerifyFailure(t *testing.T) {
	store := idempotency.NewMemoryStore()
	mockAdapter := NewMockAdapter()

	// 设置发送失败
	mockAdapter.SetSendError(errors.New("send failed"))

	runner := NewWorkflowRunner(store, mockAdapter)

	taskID := protocol.GenerateTaskID()
	dedupeKey := "e2e_failure_test"
	conversationID := "conv_test_001"
	content := "Test message"

	// 执行工作流
	err := runner.RunSendVerifyWorkflow(taskID, dedupeKey, conversationID, content)
	if err != nil {
		t.Fatalf("Workflow should not return error on send failure: %v", err)
	}

	// 验证最终状态为 failed
	retrieved, err := store.GetRecord(taskID)
	if err != nil {
		t.Fatalf("GetRecord failed: %v", err)
	}

	if retrieved.Status != "failed" {
		t.Errorf("Final status should be 'failed', got %v", retrieved.Status)
	}

	// 验证 task.failed 事件可以被创建
	failedPayload := protocol.TaskFailedPayload{
		TaskID:      taskID,
		ErrorCode:   "SEND_FAILED",
		ErrorReason: "Failed to send message",
	}

	envelope, err := protocol.NewEnvelope(
		protocol.KindEvent,
		protocol.PayloadTaskFailed,
		failedPayload,
	)
	if err != nil {
		t.Fatalf("Failed to create task.failed envelope: %v", err)
	}

	if envelope.PayloadType != protocol.PayloadTaskFailed {
		t.Errorf("Expected PayloadType task.failed, got %v", envelope.PayloadType)
	}
}
