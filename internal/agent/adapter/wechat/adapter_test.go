package wechat

import (
	"testing"

	"github.com/mazhiqiang666/GroupClaw-Desktop/internal/agent/adapter"
	"github.com/mazhiqiang666/GroupClaw-Desktop/internal/agent/windows"
	"github.com/mazhiqiang666/GroupClaw-Desktop/pkg/protocol"
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

func (m *mockBridge) FocusConversationByVision(windowHandle uintptr, strategy string, targetIndex int, waitAfterClickMs int) (windows.VisionFocusResult, adapter.Result) {
	// 返回一个默认的失败结果，模拟视觉Focus失败，让测试走旧路径
	return windows.VisionFocusResult{
		WindowHandle:   windowHandle,
		TargetIndex:    targetIndex,
		ClickStrategy:  strategy,
		FocusSucceeded: false,
		FocusConfidence: 0.3,
	}, adapter.Result{
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

// ==================== Basic Flow Tests (Minimum Required) ====================

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
