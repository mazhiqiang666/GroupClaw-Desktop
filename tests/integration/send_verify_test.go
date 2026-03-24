package integration

import (
	"errors"
	"testing"
	"time"

	"github.com/mazhiqiang666/GroupClaw-Desktop/internal/agent/adapter"
	"github.com/mazhiqiang666/GroupClaw-Desktop/internal/agent/idempotency"
	"github.com/mazhiqiang666/GroupClaw-Desktop/pkg/protocol"
)

// MockAdapter 模拟聊天适配器用于测试
type MockAdapter struct {
	sendCalls    []SendCall
	verifyCalls  []VerifyCall
	sendError    error
	verifyResult *protocol.MessageObs
	verifyError  error
}

type SendCall struct {
	Conv    protocol.ConversationRef
	Content string
	TaskID  string
}

type VerifyCall struct {
	Conv     protocol.ConversationRef
	Content  string
	Timeout  time.Duration
}

func NewMockAdapter() *MockAdapter {
	return &MockAdapter{
		sendCalls:   make([]SendCall, 0),
		verifyCalls: make([]VerifyCall, 0),
	}
}

func (m *MockAdapter) Name() string {
	return "mock-adapter"
}

func (m *MockAdapter) Version() string {
	return "1.0.0"
}

func (m *MockAdapter) SupportedApps() []string {
	return []string{"wechat", "qq"}
}

func (m *MockAdapter) Init(config adapter.Config) adapter.Result {
	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *MockAdapter) Destroy() adapter.Result {
	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *MockAdapter) IsAvailable() adapter.Result {
	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *MockAdapter) Detect() ([]protocol.AppInstanceRef, adapter.Result) {
	return []protocol.AppInstanceRef{
		{
			AppID:      "wechat",
			InstanceID: "instance_001",
		},
	}, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *MockAdapter) Scan(instance protocol.AppInstanceRef) ([]protocol.ConversationRef, adapter.Result) {
	convs := []protocol.ConversationRef{
		{
			HostWindowHandle: 12345,
			AppInstance:      instance,
			DisplayName:      "Test User",
			PreviewText:      "Hello",
			AvatarHash:       "abc123",
			ListPosition:     0,
		},
	}
	return convs, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *MockAdapter) Focus(conv protocol.ConversationRef) adapter.Result {
	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *MockAdapter) Read(conv protocol.ConversationRef, limit int) ([]protocol.MessageObs, adapter.Result) {
	msgs := []protocol.MessageObs{
		{
			MessageID:      "msg_001",
			ConversationID: "conv_001",
			SenderSide:     "customer",
			NormalizedText: "Hello",
			Timestamp:      time.Now(),
			ObservedAt:     time.Now(),
			MessageFingerprint: "fp_001",
		},
	}
	return msgs, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *MockAdapter) Send(conv protocol.ConversationRef, content string, taskID string) adapter.Result {
	m.sendCalls = append(m.sendCalls, SendCall{
		Conv:    conv,
		Content: content,
		TaskID:  taskID,
	})

	if m.sendError != nil {
		return adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonSendFailed,
			Error:      m.sendError.Error(),
		}
	}

	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *MockAdapter) Verify(conv protocol.ConversationRef, content string, timeout time.Duration) (*protocol.MessageObs, adapter.Result) {
	m.verifyCalls = append(m.verifyCalls, VerifyCall{
		Conv:    conv,
		Content: content,
		Timeout: timeout,
	})

	if m.verifyError != nil {
		return nil, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonVerifyFailed,
			Error:      m.verifyError.Error(),
		}
	}

	if m.verifyResult != nil {
		return m.verifyResult, adapter.Result{
			Status:     adapter.StatusSuccess,
			ReasonCode: adapter.ReasonOK,
		}
	}

	// 默认返回成功验证
	return &protocol.MessageObs{
		MessageID:          "msg_verified",
		ConversationID:     conv.DisplayName,
		SenderSide:         "customer",
		NormalizedText:     content,
		Timestamp:          time.Now(),
		ObservedAt:         time.Now(),
		MessageFingerprint: "fp_verified",
	}, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *MockAdapter) CaptureDiagnostics() (map[string]string, adapter.Result) {
	return map[string]string{
		"adapter": "mock",
		"version": "1.0.0",
	}, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *MockAdapter) GetSendCalls() []SendCall {
	return m.sendCalls
}

func (m *MockAdapter) GetVerifyCalls() []VerifyCall {
	return m.verifyCalls
}

func (m *MockAdapter) SetSendError(err error) {
	m.sendError = err
}

func (m *MockAdapter) SetVerifyResult(result *protocol.MessageObs) {
	m.verifyResult = result
}

func (m *MockAdapter) SetVerifyError(err error) {
	m.verifyError = err
}

// WorkflowRunner 工作流执行器
type WorkflowRunner struct {
	store   idempotency.Store
	adapter adapter.ChatAdapter
}

func NewWorkflowRunner(store idempotency.Store, adapter adapter.ChatAdapter) *WorkflowRunner {
	return &WorkflowRunner{
		store:   store,
		adapter: adapter,
	}
}

func (w *WorkflowRunner) RunSendVerifyWorkflow(taskID, dedupeKey, conversationID, content string) error {
	// 1. 创建记录
	record := idempotency.Record{
		TaskID:       taskID,
		DedupeKey:    dedupeKey,
		Conversation: conversationID,
		Content:      content,
		Status:       "pending",
	}
	if err := w.store.CreateRecord(record); err != nil {
		return err
	}

	// 2. 更新为发送中
	if err := w.store.UpdateRecord(taskID, idempotency.NewRecordUpdate().WithStatus("sending")); err != nil {
		return err
	}

	// 3. 调用适配器发送
	conv := protocol.ConversationRef{
		DisplayName: conversationID,
		AppInstance: protocol.AppInstanceRef{
			AppID:      "wechat",
			InstanceID: "instance_001",
		},
	}
	sendResult := w.adapter.Send(conv, content, taskID)
	if sendResult.Status != adapter.StatusSuccess {
		return w.store.UpdateRecord(taskID, idempotency.NewRecordUpdate().
			WithStatus("failed"))
	}

	// 4. 更新为已发送未验证
	if err := w.store.UpdateRecord(taskID, idempotency.NewRecordUpdate().
		WithStatus("sent_unverified").
		WithMessageID("msg_" + taskID).
		WithFingerprint("fp_" + taskID)); err != nil {
		return err
	}

	// 5. 调用适配器验证
	verifyResult, verifyAdapterResult := w.adapter.Verify(conv, content, 5*time.Second)
	if verifyAdapterResult.Status != adapter.StatusSuccess {
		// 验证失败，更新为未知投递状态
		return w.store.UpdateRecord(taskID, idempotency.NewRecordUpdate().
			WithStatus("unknown_delivery_state").
			WithVerifyStatus("timeout").
			WithVerifyCount(3))
	}

	// 6. 验证成功，更新为已验证
	if err := w.store.UpdateRecord(taskID, idempotency.NewRecordUpdate().
		WithStatus("verified").
		WithVerifyStatus("success").
		WithVerifyCount(1).
		WithFingerprint(verifyResult.MessageFingerprint)); err != nil {
		return err
	}

	return nil
}

// TestSendVerifyTimeoutUnknownDeliveryState 测试发送 -> 验证超时 -> 未知投递状态 -> 同任务被阻塞的流程
func TestSendVerifyTimeoutUnknownDeliveryState(t *testing.T) {
	store := idempotency.NewMemoryStore()

	// 1. 创建发送任务
	taskID := protocol.GenerateTaskID()
	dedupeKey := "dedupe_test_001"

	record := idempotency.Record{
		TaskID:       taskID,
		DedupeKey:    dedupeKey,
		Conversation: "conv_001",
		Content:      "测试消息",
		Status:       "pending",
	}

	err := store.CreateRecord(record)
	if err != nil {
		t.Fatalf("CreateRecord failed: %v", err)
	}

	// 2. 模拟发送中状态
	err = store.UpdateRecord(taskID, idempotency.NewRecordUpdate().WithStatus("sending"))
	if err != nil {
		t.Fatalf("UpdateRecord (sending) failed: %v", err)
	}

	// 验证状态
	retrieved, err := store.GetRecord(taskID)
	if err != nil {
		t.Fatalf("GetRecord failed: %v", err)
	}
	if retrieved.Status != "sending" {
		t.Errorf("Status should be 'sending', got %v", retrieved.Status)
	}

	// 3. 模拟验证超时 -> unknown_delivery_state
	err = store.UpdateRecord(taskID, idempotency.NewRecordUpdate().
		WithStatus("unknown_delivery_state").
		WithVerifyCount(3).
		WithVerifyStatus("timeout"))
	if err != nil {
		t.Fatalf("UpdateRecord (unknown_delivery_state) failed: %v", err)
	}

	// 验证状态
	retrieved, err = store.GetRecord(taskID)
	if err != nil {
		t.Fatalf("GetRecord failed: %v", err)
	}
	if retrieved.Status != "unknown_delivery_state" {
		t.Errorf("Status should be 'unknown_delivery_state', got %v", retrieved.Status)
	}
	if retrieved.VerifyCount != 3 {
		t.Errorf("VerifyCount should be 3, got %v", retrieved.VerifyCount)
	}

	// 4. 检查重复任务（同任务被阻塞）
	duplicate, err := store.CheckDuplicate(dedupeKey)
	if err != nil {
		t.Fatalf("CheckDuplicate failed: %v", err)
	}

	if duplicate == nil {
		t.Error("Duplicate record should not be nil")
	}

	if duplicate.TaskID != taskID {
		t.Errorf("Duplicate TaskID mismatch: got %v, want %v", duplicate.TaskID, taskID)
	}

	// 5. 验证同任务不能再次创建
	err = store.CreateRecord(idempotency.Record{
		TaskID:       protocol.GenerateTaskID(), // 新的 task_id
		DedupeKey:    dedupeKey,                 // 相同的 dedupe_key
		Conversation: "conv_001",
		Content:      "测试消息",
		Status:       "pending",
	})

	// 应该失败，因为 dedupe_key 已存在
	if err == nil {
		t.Error("CreateRecord should fail with duplicate dedupe_key")
	}
}

// TestSendVerifySuccess 测试发送 -> 验证成功的流程
func TestSendVerifySuccess(t *testing.T) {
	store := idempotency.NewMemoryStore()

	// 1. 创建发送任务
	taskID := protocol.GenerateTaskID()

	record := idempotency.Record{
		TaskID:       taskID,
		DedupeKey:    "dedupe_test_002",
		Conversation: "conv_001",
		Content:      "测试消息",
		Status:       "pending",
	}

	err := store.CreateRecord(record)
	if err != nil {
		t.Fatalf("CreateRecord failed: %v", err)
	}

	// 2. 模拟发送成功
	err = store.UpdateRecord(taskID, idempotency.NewRecordUpdate().
		WithStatus("sent_unverified").
		WithMessageID("msg_test_001").
		WithFingerprint("fp_test_001").
		WithVerifyCount(0))
	if err != nil {
		t.Fatalf("UpdateRecord (sent_unverified) failed: %v", err)
	}

	// 3. 模拟验证成功
	err = store.UpdateRecord(taskID, idempotency.NewRecordUpdate().
		WithStatus("verified").
		WithVerifyStatus("success").
		WithVerifyCount(1))
	if err != nil {
		t.Fatalf("UpdateRecord (verified) failed: %v", err)
	}

	// 验证最终状态
	retrieved, err := store.GetRecord(taskID)
	if err != nil {
		t.Fatalf("GetRecord failed: %v", err)
	}

	if retrieved.Status != "verified" {
		t.Errorf("Status should be 'verified', got %v", retrieved.Status)
	}

	if retrieved.VerifyStatus != "success" {
		t.Errorf("VerifyStatus should be 'success', got %v", retrieved.VerifyStatus)
	}

	if retrieved.VerifyCount != 1 {
		t.Errorf("VerifyCount should be 1, got %v", retrieved.VerifyCount)
	}
}

// TestMockAdapter_SendVerifyWorkflow 测试使用 MockAdapter 的完整工作流
func TestMockAdapter_SendVerifyWorkflow(t *testing.T) {
	store := idempotency.NewMemoryStore()
	mockAdapter := NewMockAdapter()
	runner := NewWorkflowRunner(store, mockAdapter)

	taskID := protocol.GenerateTaskID()
	dedupeKey := "mock_workflow_test"
	conversationID := "conv_mock_001"
	content := "Mock workflow test message"

	// 执行工作流
	err := runner.RunSendVerifyWorkflow(taskID, dedupeKey, conversationID, content)
	if err != nil {
		t.Fatalf("Workflow failed: %v", err)
	}

	// 验证最终状态
	retrieved, err := store.GetRecord(taskID)
	if err != nil {
		t.Fatalf("GetRecord failed: %v", err)
	}

	if retrieved.Status != "verified" {
		t.Errorf("Final status should be 'verified', got %v", retrieved.Status)
	}

	if retrieved.VerifyStatus != "success" {
		t.Errorf("VerifyStatus should be 'success', got %v", retrieved.VerifyStatus)
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

// TestMockAdapter_VerifyTimeout 测试验证超时场景
func TestMockAdapter_VerifyTimeout(t *testing.T) {
	store := idempotency.NewMemoryStore()
	mockAdapter := NewMockAdapter()

	// 设置验证返回错误（模拟超时）
	mockAdapter.SetVerifyError(errors.New("verification timeout"))

	runner := NewWorkflowRunner(store, mockAdapter)

	taskID := protocol.GenerateTaskID()
	dedupeKey := "mock_timeout_test"
	conversationID := "conv_timeout_001"
	content := "Timeout test message"

	// 执行工作流
	err := runner.RunSendVerifyWorkflow(taskID, dedupeKey, conversationID, content)
	if err != nil {
		t.Fatalf("Workflow failed: %v", err)
	}

	// 验证最终状态为 unknown_delivery_state
	retrieved, err := store.GetRecord(taskID)
	if err != nil {
		t.Fatalf("GetRecord failed: %v", err)
	}

	if retrieved.Status != "unknown_delivery_state" {
		t.Errorf("Final status should be 'unknown_delivery_state', got %v", retrieved.Status)
	}

	if retrieved.VerifyStatus != "timeout" {
		t.Errorf("VerifyStatus should be 'timeout', got %v", retrieved.VerifyStatus)
	}

	if retrieved.VerifyCount != 3 {
		t.Errorf("VerifyCount should be 3, got %v", retrieved.VerifyCount)
	}
}
