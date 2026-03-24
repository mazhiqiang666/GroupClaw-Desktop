package wechat

import (
	"testing"
	"time"

	"github.com/mazhiqiang666/GroupClaw-Desktop/internal/agent/adapter"
	"github.com/mazhiqiang666/GroupClaw-Desktop/internal/agent/windows"
)

// controlledMockBridge 是一个用于最小闭环测试的 mock bridge，提供完全控制
type controlledMockBridge struct {
	initialized    bool
	findResult     []uintptr
	findError      adapter.Result
	windowClass    string
	windowTitle    string
	enumerateError adapter.Result
	// Controlled nodes for testing
	nodes []windows.AccessibleNode
	// Track calls for verification
	focusCalled      bool
	sendCalled       bool
	verifyCalled     bool
	lastFocusHandle  uintptr
	lastSendContent  string
	lastSendTaskID   string
	sendKeysCalls    []string  // 记录所有 SendKeys 调用
	clipboardContent string    // 记录剪贴板内容
}

func newControlledMockBridge() *controlledMockBridge {
	return &controlledMockBridge{
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
		nodes: []windows.AccessibleNode{
			{
				Handle:    1,
				Name:      "张三",
				Role:      "list item",
				ClassName: "",
				Bounds:    [4]int{10, 50, 180, 40},
				Children:  []windows.AccessibleNode{},
				TreePath:  "[0]",
			},
			{
				Handle:    2,
				Name:      "李四",
				Role:      "list item",
				ClassName: "",
				Bounds:    [4]int{10, 90, 180, 40},
				Children:  []windows.AccessibleNode{},
				TreePath:  "[1]",
			},
		},
	}
}

func (m *controlledMockBridge) Initialize() adapter.Result {
	m.initialized = true
	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *controlledMockBridge) FindTopLevelWindows(className, windowName string) ([]uintptr, adapter.Result) {
	if m.findError.Status != adapter.StatusSuccess {
		return nil, m.findError
	}
	return m.findResult, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *controlledMockBridge) FindWindow(className, windowName string) (uintptr, adapter.Result) {
	return 0, adapter.Result{
		Status:     adapter.StatusFailed,
		ReasonCode: adapter.ReasonCode("NOT_IMPLEMENTED"),
	}
}

func (m *controlledMockBridge) FindChildWindow(parentHandle uintptr, className, windowName string) (uintptr, adapter.Result) {
	return 0, adapter.Result{
		Status:     adapter.StatusFailed,
		ReasonCode: adapter.ReasonCode("NOT_IMPLEMENTED"),
	}
}

func (m *controlledMockBridge) GetWindowText(handle uintptr) (string, adapter.Result) {
	return "", adapter.Result{
		Status:     adapter.StatusFailed,
		ReasonCode: adapter.ReasonCode("NOT_IMPLEMENTED"),
	}
}

func (m *controlledMockBridge) GetWindowClass(handle uintptr) (string, adapter.Result) {
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

func (m *controlledMockBridge) GetWindowInfo(handle uintptr) (windows.WindowInfo, adapter.Result) {
	return windows.WindowInfo{
		Handle: handle,
		Class:  m.windowClass,
		Title:  m.windowTitle,
	}, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *controlledMockBridge) FocusWindow(handle uintptr) adapter.Result {
	m.focusCalled = true
	m.lastFocusHandle = handle
	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *controlledMockBridge) EnumerateAccessibleNodes(windowHandle uintptr) ([]windows.AccessibleNode, adapter.Result) {
	if m.enumerateError.Status != adapter.StatusSuccess {
		return nil, m.enumerateError
	}
	return m.nodes, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *controlledMockBridge) GetAccessible(windowHandle uintptr) (*windows.IAccessible, adapter.Result) {
	return nil, adapter.Result{
		Status:     adapter.StatusFailed,
		ReasonCode: adapter.ReasonCode("NOT_IMPLEMENTED"),
	}
}

func (m *controlledMockBridge) CaptureWindow(handle uintptr) ([]byte, adapter.Result) {
	return []byte{}, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *controlledMockBridge) SendKeys(handle uintptr, keys string) adapter.Result {
	m.sendCalled = true
	m.lastSendContent = keys
	m.sendKeysCalls = append(m.sendKeysCalls, keys)
	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *controlledMockBridge) Click(handle uintptr, x, y int) adapter.Result {
	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *controlledMockBridge) SetClipboardText(text string) adapter.Result {
	m.clipboardContent = text
	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *controlledMockBridge) GetClipboardText() (string, adapter.Result) {
	return "", adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *controlledMockBridge) DetectConversations(windowHandle uintptr) (windows.VisionDebugResult, adapter.Result) {
	// 返回一个模拟的视觉检测结果，包含2个会话矩形
	rects := []windows.ConversationRect{
		{
			Index:        0,
			X:            10,
			Y:            50,
			Width:        180,
			Height:       40,
			HasAvatar:    true,
			HasText:      true,
			HasUnreadDot: false,
			IsSelected:   false,
			AvatarRect:   [4]int{15, 55, 30, 30},
			TextRect:     [4]int{55, 60, 120, 25},
		},
		{
			Index:        1,
			X:            10,
			Y:            90,
			Width:        180,
			Height:       40,
			HasAvatar:    true,
			HasText:      true,
			HasUnreadDot: true,
			IsSelected:   false,
			AvatarRect:   [4]int{15, 95, 30, 30},
			TextRect:     [4]int{55, 100, 120, 25},
			UnreadDotRect: [4]int{5, 100, 8, 8},
		},
	}
	features := map[string]int{
		"conversation_rects": 2,
		"avatars": 2,
		"text_regions": 2,
		"unread_dots": 1,
	}
	return windows.VisionDebugResult{
		WindowHandle:     windowHandle,
		WindowWidth:      800,
		WindowHeight:     600,
		ImageSize:        1000,
		LeftSidebarRect:  [4]int{0, 0, 200, 600},
		ConversationRects: rects,
		DetectedFeatures: features,
		ProcessingTime:   100 * time.Millisecond,
	}, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *controlledMockBridge) FocusConversationByVision(windowHandle uintptr, strategy string, targetIndex int, waitAfterClickMs int) (windows.VisionFocusResult, adapter.Result) {
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

func (m *controlledMockBridge) Release() {
	m.initialized = false
}

// ==================== Minimum Closed-Loop Diagnostic Test ====================

func TestWeChatAdapter_MinimumClosedLoop_Diagnostics(t *testing.T) {
	mock := newControlledMockBridge()
	mock.findResult = []uintptr{12345}
	wechatAdapter := NewWeChatAdapterWithBridge(mock)

	// Step 1: Detect instances
	instances, detectResult := wechatAdapter.Detect()
	if detectResult.Status != adapter.StatusSuccess {
		t.Fatalf("Detect should succeed, got status: %v", detectResult.Status)
	}
	if len(instances) == 0 {
		t.Fatal("Detect should find at least one instance")
	}

	// Step 2: Scan conversations
	conversations, scanResult := wechatAdapter.Scan(instances[0])
	if scanResult.Status != adapter.StatusSuccess {
		t.Fatalf("Scan should succeed, got status: %v", scanResult.Status)
	}
	if len(conversations) == 0 {
		t.Fatal("Scan should return at least one conversation")
	}

	// Step 3: Focus on conversation
	conv := conversations[0]
	focusResult := wechatAdapter.Focus(conv)
	if focusResult.Status != adapter.StatusSuccess {
		t.Fatalf("Focus should succeed, got status: %v", focusResult.Status)
	}
	if !mock.focusCalled {
		t.Error("Focus should have called bridge.FocusWindow")
	}

	// Step 4: Send message
	sendResult := wechatAdapter.Send(conv, "Test message", "task-123")
	if sendResult.Status != adapter.StatusSuccess {
		t.Fatalf("Send should succeed, got status: %v", sendResult.Status)
	}
	if !mock.sendCalled {
		t.Error("Send should have called bridge.SendKeys")
	}
	// Check that the message was set to clipboard
	if mock.clipboardContent != "Test message" {
		t.Errorf("Clipboard content mismatch: expected 'Test message', got '%s'", mock.clipboardContent)
	}
	// Check that SendKeys was called with paste and enter commands
	if len(mock.sendKeysCalls) < 2 {
		t.Errorf("Expected at least 2 SendKeys calls (paste + enter), got %d", len(mock.sendKeysCalls))
	} else {
		// First call should be paste (^v)
		if mock.sendKeysCalls[0] != "^v" {
			t.Errorf("First SendKeys call should be '^v', got '%s'", mock.sendKeysCalls[0])
		}
		// Second call should be enter ({ENTER})
		if mock.sendKeysCalls[1] != "{ENTER}" {
			t.Errorf("Second SendKeys call should be '{{ENTER}}', got '%s'", mock.sendKeysCalls[1])
		}
	}

	// Step 5: Verify message delivery
	msg, verifyResult := wechatAdapter.Verify(conv, "Test message", 5)
	if verifyResult.Status != adapter.StatusSuccess {
		t.Fatalf("Verify should succeed, got status: %v", verifyResult.Status)
	}
	// Note: In stub implementation, msg will be nil
	if msg != nil {
		t.Log("Verify returned a message (stub implementation may vary)")
	}

	// Verify diagnostics structure from focus operation
	// The focus operation should produce diagnostics with locate_source, evidence_count, confidence
	// We can't directly access the diagnostics from the result, but we can verify the
	// conversion functions work correctly

	// Test ConvertFocusEvidenceToDiagnostics
	evidence := FocusVerificationEvidence{
		LocateSource:         "tree_path_name",
		NodeStillExists:      true,
		NodeHasActiveState:   true,
		TitleContainsTarget:  false,
		PanelSwitchDetected:  false,
		MessageAreaVisible:   true,
		Confidence:           0.85,
		EvidenceCount:        3,
	}

	diagnostics := ConvertFocusEvidenceToDiagnostics(evidence)

	// Verify required diagnostic fields
	if diagnostics["locate_source"] != "tree_path_name" {
		t.Errorf("locate_source mismatch: expected 'tree_path_name', got '%s'", diagnostics["locate_source"])
	}
	if diagnostics["evidence_count"] != "3" {
		t.Errorf("evidence_count mismatch: expected '3', got '%s'", diagnostics["evidence_count"])
	}
	if diagnostics["confidence"] != "0.85" {
		t.Errorf("confidence mismatch: expected '0.85', got '%s'", diagnostics["confidence"])
	}

	// Test ConvertMessageEvidenceToDiagnostics
	messageEvidence := SendVerificationEvidence{
		NewMessageNodes:     1,
		MessageNodeAdded:    true,
		MessageContentMatch: true,
		ScreenshotChanged:   true,
		ChatAreaDiff:        0.05,
		Confidence:          0.9,
	}

	messageDiagnostics := ConvertMessageEvidenceToDiagnostics(messageEvidence)
	if messageDiagnostics["confidence"] != "0.90" {
		t.Errorf("message confidence mismatch: expected '0.90', got '%s'", messageDiagnostics["confidence"])
	}

	// Test ConvertDeliveryAssessmentToDiagnostics
	assessment := DeliveryAssessment{
		State:      "verified",
		Confidence: 0.85,
		Messages:   []string{"Test message delivered"},
	}

	deliveryDiagnostics := ConvertDeliveryAssessmentToDiagnostics(assessment)
	if deliveryDiagnostics["delivery_state"] != "verified" {
		t.Errorf("delivery_state mismatch: expected 'verified', got '%s'", deliveryDiagnostics["delivery_state"])
	}
	if deliveryDiagnostics["confidence"] != "0.85" {
		t.Errorf("delivery confidence mismatch: expected '0.85', got '%s'", deliveryDiagnostics["confidence"])
	}
}

func TestWeChatAdapter_MinimumClosedLoop_WithControlledNodes(t *testing.T) {
	mock := newControlledMockBridge()
	mock.findResult = []uintptr{12345}

	// Set up controlled nodes with specific properties
	mock.nodes = []windows.AccessibleNode{
		{
			Handle:    100,
			Name:      "Test Conversation",
			Role:      "list item",
			ClassName: "",
			Bounds:    [4]int{10, 50, 180, 40},
			Children:  []windows.AccessibleNode{},
			TreePath:  "[0]",
		},
		{
			Handle:    101,
			Name:      "Message Area",
			Role:      "text",
			ClassName: "",
			Bounds:    [4]int{200, 100, 300, 200},
			Children:  []windows.AccessibleNode{},
			TreePath:  "[1]",
		},
	}

	wechatAdapter := NewWeChatAdapterWithBridge(mock)

	// Detect and Scan
	instances, _ := wechatAdapter.Detect()
	conversations, _ := wechatAdapter.Scan(instances[0])

	if len(conversations) == 0 {
		t.Fatal("Expected at least one conversation with controlled nodes")
	}

	// Verify conversation properties
	conv := conversations[0]
	// With vision scan, DisplayName will be "conversation_0" instead of "Test Conversation"
	expectedName := "conversation_0"
	if conv.DisplayName != expectedName {
		t.Errorf("Expected conversation name '%s', got '%s'", expectedName, conv.DisplayName)
	}

	// Focus on the controlled conversation
	focusResult := wechatAdapter.Focus(conv)
	if focusResult.Status != adapter.StatusSuccess {
		t.Fatalf("Focus failed: %v", focusResult.Status)
	}

	// Send message to controlled conversation
	sendResult := wechatAdapter.Send(conv, "Controlled test message", "task-456")
	if sendResult.Status != adapter.StatusSuccess {
		t.Fatalf("Send failed: %v", sendResult.Status)
	}

	// Verify the controlled nodes were used
	if !mock.focusCalled {
		t.Error("Focus should have been called on controlled nodes")
	}
	if !mock.sendCalled {
		t.Error("Send should have been called on controlled nodes")
	}

	// Verify diagnostic consistency
	// Create a sample evidence and verify it converts to expected diagnostics
	evidence := FocusVerificationEvidence{
		LocateSource:       "controlled_test",
		NodeStillExists:    true,
		NodeHasActiveState: true,
		Confidence:         0.95,
		EvidenceCount:      5,
	}

	diagnostics := ConvertFocusEvidenceToDiagnostics(evidence)

	// Verify all required fields are present
	requiredFields := []string{"locate_source", "evidence_count", "confidence", "node_still_exists"}
	for _, field := range requiredFields {
		if _, ok := diagnostics[field]; !ok {
			t.Errorf("Missing required diagnostic field: %s", field)
		}
	}

	// Verify confidence format (should be 2 decimal places)
	if diagnostics["confidence"] != "0.95" {
		t.Errorf("Confidence format incorrect: expected '0.95', got '%s'", diagnostics["confidence"])
	}
}
