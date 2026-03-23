package wechat

import (
	"testing"

	"github.com/yourorg/auto-customer-service/internal/agent/adapter"
	"github.com/yourorg/auto-customer-service/internal/agent/windows"
	"github.com/yourorg/auto-customer-service/pkg/protocol"
)

// mockBridge 是一个用于测试的 mock bridge 实现
type mockBridge struct {
	initialized    bool
	findResult     []uintptr
	findError      adapter.Result
	windowClass    string
	windowTitle    string
	enumerateError adapter.Result
}

func newMockBridge() *mockBridge {
	return &mockBridge{
		findError: adapter.Result{
			Status:     adapter.StatusSuccess,
			ReasonCode: adapter.ReasonOK,
		},
		windowClass: "WeChatMainWndForPC",
		windowTitle: "微信",
		enumerateError: adapter.Result{
			Status:     adapter.StatusSuccess,
			ReasonCode: adapter.ReasonOK,
		},
	}
}

func (m *mockBridge) Initialize() adapter.Result {
	m.initialized = true
	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *mockBridge) FindTopLevelWindows(className, windowName string) ([]uintptr, adapter.Result) {
	if m.findError.Status != adapter.StatusSuccess {
		return nil, m.findError
	}
	return m.findResult, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *mockBridge) FindWindow(className, windowName string) (uintptr, adapter.Result) {
	return 0, adapter.Result{
		Status:     adapter.StatusFailed,
		ReasonCode: adapter.ReasonCode("NOT_IMPLEMENTED"),
	}
}

func (m *mockBridge) FindChildWindow(parentHandle uintptr, className, windowName string) (uintptr, adapter.Result) {
	return 0, adapter.Result{
		Status:     adapter.StatusFailed,
		ReasonCode: adapter.ReasonCode("NOT_IMPLEMENTED"),
	}
}

func (m *mockBridge) GetWindowText(handle uintptr) (string, adapter.Result) {
	return "", adapter.Result{
		Status:     adapter.StatusFailed,
		ReasonCode: adapter.ReasonCode("NOT_IMPLEMENTED"),
	}
}

func (m *mockBridge) GetWindowClass(handle uintptr) (string, adapter.Result) {
	if m.windowClass == "" {
		return "", adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("NOT_FOUND"),
		}
	}
	return m.windowClass, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *mockBridge) GetWindowInfo(handle uintptr) (windows.WindowInfo, adapter.Result) {
	return windows.WindowInfo{
		Handle: handle,
		Class:  m.windowClass,
		Title:  m.windowTitle,
	}, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *mockBridge) FocusWindow(handle uintptr) adapter.Result {
	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *mockBridge) EnumerateAccessibleNodes(windowHandle uintptr) ([]windows.AccessibleNode, adapter.Result) {
	if m.enumerateError.Status != adapter.StatusSuccess {
		return nil, m.enumerateError
	}
	nodes := []windows.AccessibleNode{
		{
			Handle: 1,
			Name:   "张三",
			Role:   "list item",
			Bounds: [4]int{10, 50, 180, 40}, // x, y, width, height - within left 1/3 area
		},
		{
			Handle: 2,
			Name:   "李四",
			Role:   "list item",
			Bounds: [4]int{10, 90, 180, 40}, // x, y, width, height - within left 1/3 area
		},
	}
	return nodes, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *mockBridge) GetAccessible(windowHandle uintptr) (*windows.IAccessible, adapter.Result) {
	return nil, adapter.Result{
		Status:     adapter.StatusFailed,
		ReasonCode: adapter.ReasonCode("NOT_IMPLEMENTED"),
	}
}

func (m *mockBridge) CaptureWindow(handle uintptr) ([]byte, adapter.Result) {
	return []byte{}, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *mockBridge) SendKeys(handle uintptr, keys string) adapter.Result {
	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *mockBridge) Click(handle uintptr, x, y int) adapter.Result {
	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *mockBridge) SetClipboardText(text string) adapter.Result {
	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *mockBridge) GetClipboardText() (string, adapter.Result) {
	return "", adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *mockBridge) Release() {
	m.initialized = false
}

func TestNewWeChatAdapter(t *testing.T) {
	wechatAdapter := NewWeChatAdapter()
	if wechatAdapter == nil {
		t.Error("NewWeChatAdapter should return a non-nil adapter")
	}
	if wechatAdapter.Name() != "wechat" {
		t.Errorf("Expected adapter name 'wechat', got '%s'", wechatAdapter.Name())
	}
}

func TestNewWeChatAdapterWithBridge(t *testing.T) {
	mock := newMockBridge()
	wechatAdapter := NewWeChatAdapterWithBridge(mock)
	if wechatAdapter == nil {
		t.Error("NewWeChatAdapterWithBridge should return a non-nil adapter")
	}
}

func TestWeChatAdapter_Init(t *testing.T) {
	mock := newMockBridge()
	wechatAdapter := NewWeChatAdapterWithBridge(mock)

	config := adapter.Config{}
	result := wechatAdapter.Init(config)

	if result.Status != adapter.StatusSuccess {
		t.Errorf("Init should succeed, got status: %v", result.Status)
	}

	if !mock.initialized {
		t.Error("Bridge should be initialized after Init")
	}
}

func TestWeChatAdapter_Detect(t *testing.T) {
	mock := newMockBridge()
	mock.findResult = []uintptr{12345}
	wechatAdapter := NewWeChatAdapterWithBridge(mock)

	instances, result := wechatAdapter.Detect()

	if result.Status != adapter.StatusSuccess {
		t.Errorf("Detect should succeed, got status: %v", result.Status)
	}

	if len(instances) != 1 {
		t.Errorf("Expected 1 instance, got %d", len(instances))
	}

	if len(instances) > 0 {
		if instances[0].AppID != "wechat" {
			t.Errorf("Expected AppID 'wechat', got '%s'", instances[0].AppID)
		}
		if instances[0].InstanceID != "微信" {
			t.Errorf("Expected InstanceID '微信', got '%s'", instances[0].InstanceID)
		}
	}
}

func TestWeChatAdapter_Detect_NoWindow(t *testing.T) {
	mock := newMockBridge()
	mock.findResult = []uintptr{} // 没有找到窗口
	wechatAdapter := NewWeChatAdapterWithBridge(mock)

	instances, result := wechatAdapter.Detect()

	if result.Status != adapter.StatusSuccess {
		t.Errorf("Detect should succeed even with no windows, got status: %v", result.Status)
	}

	if len(instances) != 0 {
		t.Errorf("Expected 0 instances, got %d", len(instances))
	}
}

func TestWeChatAdapter_Detect_MultipleWindows(t *testing.T) {
	mock := newMockBridge()
	mock.findResult = []uintptr{12345, 67890}
	wechatAdapter := NewWeChatAdapterWithBridge(mock)

	instances, result := wechatAdapter.Detect()

	if result.Status != adapter.StatusSuccess {
		t.Errorf("Detect should succeed, got status: %v", result.Status)
	}

	if len(instances) != 2 {
		t.Errorf("Expected 2 instances, got %d", len(instances))
	}
}

func TestWeChatAdapter_Detect_BridgeError(t *testing.T) {
	mock := newMockBridge()
	mock.findError = adapter.Result{
		Status:     adapter.StatusFailed,
		ReasonCode: adapter.ReasonCode("BRIDGE_ERROR"),
		Error:      "Bridge error occurred",
	}
	wechatAdapter := NewWeChatAdapterWithBridge(mock)

	instances, result := wechatAdapter.Detect()

	if result.Status != adapter.StatusFailed {
		t.Errorf("Detect should fail when bridge fails, got status: %v", result.Status)
	}

	if instances != nil {
		t.Error("Detect should return nil instances on bridge error")
	}
}

func TestWeChatAdapter_Scan(t *testing.T) {
	mock := newMockBridge()
	mock.findResult = []uintptr{12345}
	wechatAdapter := NewWeChatAdapterWithBridge(mock)

	instance := protocol.AppInstanceRef{
		AppID:      "wechat",
		InstanceID: "微信",
	}

	conversations, result := wechatAdapter.Scan(instance)

	if result.Status != adapter.StatusSuccess {
		t.Errorf("Scan should succeed, got status: %v", result.Status)
	}

	if len(conversations) != 2 {
		t.Errorf("Expected 2 conversations, got %d", len(conversations))
	}

	if len(conversations) > 0 {
		if conversations[0].DisplayName != "张三" {
			t.Errorf("Expected first conversation name '张三', got '%s'", conversations[0].DisplayName)
		}
		if conversations[0].HostWindowHandle != 12345 {
			t.Errorf("Expected HostWindowHandle 12345, got %d", conversations[0].HostWindowHandle)
		}
	}
}

func TestWeChatAdapter_Scan_NoWindow(t *testing.T) {
	mock := newMockBridge()
	mock.findResult = []uintptr{} // 没有找到窗口
	wechatAdapter := NewWeChatAdapterWithBridge(mock)

	instance := protocol.AppInstanceRef{
		AppID:      "wechat",
		InstanceID: "微信",
	}

	conversations, result := wechatAdapter.Scan(instance)

	if result.Status != adapter.StatusSuccess {
		t.Errorf("Scan should succeed even with no windows, got status: %v", result.Status)
	}

	if len(conversations) != 0 {
		t.Errorf("Expected 0 conversations, got %d", len(conversations))
	}
}

func TestWeChatAdapter_Scan_BridgeError(t *testing.T) {
	mock := newMockBridge()
	mock.findError = adapter.Result{
		Status:     adapter.StatusFailed,
		ReasonCode: adapter.ReasonCode("BRIDGE_ERROR"),
		Error:      "Bridge error occurred",
	}
	wechatAdapter := NewWeChatAdapterWithBridge(mock)

	instance := protocol.AppInstanceRef{
		AppID:      "wechat",
		InstanceID: "微信",
	}

	conversations, result := wechatAdapter.Scan(instance)

	if result.Status != adapter.StatusFailed {
		t.Errorf("Scan should fail when bridge fails, got status: %v", result.Status)
	}

	if conversations != nil {
		t.Error("Scan should return nil conversations on bridge error")
	}
}

func TestWeChatAdapter_Focus(t *testing.T) {
	mock := newMockBridge()
	wechatAdapter := NewWeChatAdapterWithBridge(mock)

	conv := protocol.ConversationRef{
		HostWindowHandle: 12345,
	}

	result := wechatAdapter.Focus(conv)

	if result.Status != adapter.StatusSuccess {
		t.Errorf("Focus should succeed, got status: %v", result.Status)
	}
}

func TestWeChatAdapter_Read(t *testing.T) {
	mock := newMockBridge()
	wechatAdapter := NewWeChatAdapterWithBridge(mock)

	conv := protocol.ConversationRef{
		HostWindowHandle: 12345,
	}

	messages, result := wechatAdapter.Read(conv, 10)

	if result.Status != adapter.StatusSuccess {
		t.Errorf("Read should succeed, got status: %v", result.Status)
	}

	if messages == nil {
		t.Error("Read should return a non-nil message slice")
	}
}

func TestWeChatAdapter_Send(t *testing.T) {
	mock := newMockBridge()
	wechatAdapter := NewWeChatAdapterWithBridge(mock)

	conv := protocol.ConversationRef{
		HostWindowHandle: 12345,
	}

	result := wechatAdapter.Send(conv, "Hello", "task-123")

	if result.Status != adapter.StatusSuccess {
		t.Errorf("Send should succeed, got status: %v", result.Status)
	}
}

func TestWeChatAdapter_Verify(t *testing.T) {
	mock := newMockBridge()
	wechatAdapter := NewWeChatAdapterWithBridge(mock)

	conv := protocol.ConversationRef{
		HostWindowHandle: 12345,
	}

	msg, result := wechatAdapter.Verify(conv, "Hello", 5)

	if result.Status != adapter.StatusSuccess {
		t.Errorf("Verify should succeed, got status: %v", result.Status)
	}

	// Verify returns nil message for stub implementation
	if msg != nil {
		t.Error("Verify should return nil message for stub implementation")
	}
}

func TestWeChatAdapter_Verify_EmptyContent(t *testing.T) {
	mock := newMockBridge()
	wechatAdapter := NewWeChatAdapterWithBridge(mock)

	conv := protocol.ConversationRef{
		HostWindowHandle: 12345,
	}

	_, result := wechatAdapter.Verify(conv, "", 5)

	if result.Status != adapter.StatusSuccess {
		t.Errorf("Verify should succeed even with empty content, got status: %v", result.Status)
	}
}

func TestWeChatAdapter_Verify_ZeroTimeout(t *testing.T) {
	mock := newMockBridge()
	wechatAdapter := NewWeChatAdapterWithBridge(mock)

	conv := protocol.ConversationRef{
		HostWindowHandle: 12345,
	}

	msg, result := wechatAdapter.Verify(conv, "Hello", 0)

	if result.Status != adapter.StatusSuccess {
		t.Errorf("Verify should succeed even with zero timeout, got status: %v", result.Status)
	}

	if msg != nil {
		t.Error("Verify should return nil message for stub implementation")
	}
}

func TestWeChatAdapter_CaptureDiagnostics(t *testing.T) {
	mock := newMockBridge()
	mock.findResult = []uintptr{12345}
	wechatAdapter := NewWeChatAdapterWithBridge(mock)

	diagnostics, result := wechatAdapter.CaptureDiagnostics()

	if result.Status != adapter.StatusSuccess {
		t.Errorf("CaptureDiagnostics should succeed, got status: %v", result.Status)
	}

	if diagnostics["adapter_name"] != "wechat" {
		t.Errorf("Expected adapter_name 'wechat', got '%s'", diagnostics["adapter_name"])
	}

	if diagnostics["bridge_status"] != "available" {
		t.Errorf("Expected bridge_status 'available', got '%s'", diagnostics["bridge_status"])
	}
}

func TestWeChatAdapter_IsAvailable(t *testing.T) {
	mock := newMockBridge()
	wechatAdapter := NewWeChatAdapterWithBridge(mock)

	result := wechatAdapter.IsAvailable()

	if result.Status != adapter.StatusSuccess {
		t.Errorf("IsAvailable should succeed, got status: %v", result.Status)
	}

	if result.Confidence != 1.0 {
		t.Errorf("Expected confidence 1.0, got %f", result.Confidence)
	}
}

func TestWeChatAdapter_Detect_WithClassVerification(t *testing.T) {
	mock := newMockBridge()
	mock.findResult = []uintptr{12345}
	mock.windowClass = "WeChatMainWndForPC"
	mock.windowTitle = "微信"
	wechatAdapter := NewWeChatAdapterWithBridge(mock)

	instances, result := wechatAdapter.Detect()

	if result.Status != adapter.StatusSuccess {
		t.Errorf("Detect should succeed, got status: %v", result.Status)
	}

	if len(instances) != 1 {
		t.Errorf("Expected 1 instance, got %d", len(instances))
	}

	if len(instances) > 0 {
		if instances[0].AppID != "wechat" {
			t.Errorf("Expected AppID 'wechat', got '%s'", instances[0].AppID)
		}
		if instances[0].InstanceID != "微信" {
			t.Errorf("Expected InstanceID '微信', got '%s'", instances[0].InstanceID)
		}
	}
}

func TestWeChatAdapter_Detect_NonWeChatWindow(t *testing.T) {
	mock := newMockBridge()
	mock.findResult = []uintptr{12345}
	mock.windowClass = "NotWeChatWindow"
	mock.windowTitle = "Some Other App"
	wechatAdapter := NewWeChatAdapterWithBridge(mock)

	instances, result := wechatAdapter.Detect()

	// Should return empty list when window is not WeChat
	if result.Status != adapter.StatusSuccess {
		t.Errorf("Detect should succeed, got status: %v", result.Status)
	}

	if len(instances) != 0 {
		t.Errorf("Expected 0 instances for non-WeChat window, got %d", len(instances))
	}
}

func TestWeChatAdapter_Detect_FallbackToClassName(t *testing.T) {
	mock := newMockBridge()
	mock.findResult = []uintptr{12345}
	mock.windowClass = "WeChatMainWndForPC"
	mock.windowTitle = "Some Title"
	wechatAdapter := NewWeChatAdapterWithBridge(mock)

	instances, result := wechatAdapter.Detect()

	if result.Status != adapter.StatusSuccess {
		t.Errorf("Detect should succeed, got status: %v", result.Status)
	}

	if len(instances) != 1 {
		t.Errorf("Expected 1 instance (matched by class), got %d", len(instances))
	}
}

func TestWeChatAdapter_Scan_WithPlaceholder(t *testing.T) {
	mock := newMockBridge()
	mock.findResult = []uintptr{12345}
	mock.enumerateError = adapter.Result{
		Status:     adapter.StatusFailed,
		ReasonCode: adapter.ReasonCode("ACCESSIBLE_NOT_FOUND"),
	}
	wechatAdapter := NewWeChatAdapterWithBridge(mock)

	instance := protocol.AppInstanceRef{
		AppID:      "wechat",
		InstanceID: "微信",
	}

	conversations, result := wechatAdapter.Scan(instance)

	if result.Status != adapter.StatusSuccess {
		t.Errorf("Scan should succeed even when no conversations found, got status: %v", result.Status)
	}

	// 当无法枚举节点时，返回空列表而不是占位会话
	if len(conversations) != 0 {
		t.Errorf("Expected 0 conversations when enumerate fails, got %d", len(conversations))
	}

	// 验证诊断信息包含基本上下文
	if len(result.Diagnostics) > 0 {
		diag := result.Diagnostics[0]
		if diag.Context["window_handle"] == "" {
			t.Error("Expected window_handle in diagnostics")
		}
	}
}

func TestWeChatAdapter_Scan_WithRealNodes(t *testing.T) {
	mock := newMockBridge()
	mock.findResult = []uintptr{12345}
	mock.enumerateError = adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
	wechatAdapter := NewWeChatAdapterWithBridge(mock)

	instance := protocol.AppInstanceRef{
		AppID:      "wechat",
		InstanceID: "微信",
	}

	conversations, result := wechatAdapter.Scan(instance)

	if result.Status != adapter.StatusSuccess {
		t.Errorf("Scan should succeed, got status: %v", result.Status)
	}

	if len(conversations) != 2 {
		t.Errorf("Expected 2 conversations from real nodes, got %d", len(conversations))
	}

	if len(conversations) > 0 {
		if conversations[0].DisplayName != "张三" {
			t.Errorf("Expected first conversation name '张三', got '%s'", conversations[0].DisplayName)
		}
	}
}

func TestWeChatAdapter_Integration_DetectAndScan(t *testing.T) {
	mock := newMockBridge()
	mock.findResult = []uintptr{12345}
	mock.windowClass = "WeChatMainWndForPC"
	mock.windowTitle = "微信"
	wechatAdapter := NewWeChatAdapterWithBridge(mock)

	// Step 1: Detect instances
	instances, detectResult := wechatAdapter.Detect()
	if detectResult.Status != adapter.StatusSuccess {
		t.Errorf("Detect should succeed, got status: %v", detectResult.Status)
	}

	if len(instances) == 0 {
		t.Error("Detect should find at least one instance")
		return
	}

	// Step 2: Scan conversations for first instance
	conversations, scanResult := wechatAdapter.Scan(instances[0])
	if scanResult.Status != adapter.StatusSuccess {
		t.Errorf("Scan should succeed, got status: %v", scanResult.Status)
	}

	if len(conversations) == 0 {
		t.Error("Scan should return at least one conversation (even if placeholder)")
	}
}
