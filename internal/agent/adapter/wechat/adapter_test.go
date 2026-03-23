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
			Bounds: [4]int{10, 50, 180, 40},
		},
		{
			Handle: 2,
			Name:   "李四",
			Role:   "list item",
			Bounds: [4]int{10, 90, 180, 40},
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

// ==================== Basic Flow Tests ====================

func TestWeChatAdapter_Detect(t *testing.T) {
	mock := newMockBridge()
	mock.findResult = []uintptr{12345}
	wechatAdapter := NewWeChatAdapterWithBridge(mock)

	instances, result := wechatAdapter.Detect()

	if result.Status != adapter.StatusSuccess {
		t.Errorf("Detect should succeed, got status: %v", result.Status)
	}
	if result.ReasonCode != adapter.ReasonOK {
		t.Errorf("Expected ReasonCode OK, got %v", result.ReasonCode)
	}
	if len(instances) != 1 {
		t.Errorf("Expected 1 instance, got %d", len(instances))
	}
	if len(instances) > 0 {
		if instances[0].AppID != "wechat" {
			t.Errorf("Expected AppID 'wechat', got '%s'", instances[0].AppID)
		}
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
	if result.ReasonCode != adapter.ReasonOK {
		t.Errorf("Expected ReasonCode OK, got %v", result.ReasonCode)
	}
	if len(conversations) == 0 {
		t.Error("Scan should return conversations")
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
	if result.ReasonCode != adapter.ReasonOK {
		t.Errorf("Expected ReasonCode OK, got %v", result.ReasonCode)
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
	if result.ReasonCode != adapter.ReasonOK {
		t.Errorf("Expected ReasonCode OK, got %v", result.ReasonCode)
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
	if result.ReasonCode != adapter.ReasonOK {
		t.Errorf("Expected ReasonCode OK, got %v", result.ReasonCode)
	}
	if msg != nil {
		t.Error("Verify should return nil message for stub implementation")
	}
}

// ==================== Edge Case Tests ====================

func TestWeChatAdapter_Detect_NoWindow(t *testing.T) {
	mock := newMockBridge()
	mock.findResult = []uintptr{}
	wechatAdapter := NewWeChatAdapterWithBridge(mock)

	instances, result := wechatAdapter.Detect()

	if result.Status != adapter.StatusSuccess {
		t.Errorf("Detect should succeed even with no windows, got status: %v", result.Status)
	}
	if len(instances) != 0 {
		t.Errorf("Expected 0 instances, got %d", len(instances))
	}
}

func TestWeChatAdapter_Scan_NoWindow(t *testing.T) {
	mock := newMockBridge()
	mock.findResult = []uintptr{}
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

// ==================== Integration Flow Test ====================

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
		t.Error("Scan should return at least one conversation")
	}
}
