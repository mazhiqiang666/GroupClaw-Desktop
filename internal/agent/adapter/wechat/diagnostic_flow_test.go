package wechat

import (
	"testing"
	"time"

	"github.com/yourorg/auto-customer-service/internal/agent/adapter"
	"github.com/yourorg/auto-customer-service/internal/agent/windows"
	"github.com/yourorg/auto-customer-service/pkg/protocol"
)

// stateChangingMockBridge is a mock bridge that tracks node state changes
// before/after operations to simulate real-world behavior
type stateChangingMockBridge struct {
	initialized    bool
	findResult     []uintptr
	findError      adapter.Result
	windowClass    string
	windowTitle    string
	enumerateError adapter.Result

	// Node state tracking
	nodesBeforeFocus []windows.AccessibleNode
	nodesAfterFocus  []windows.AccessibleNode
	nodesBeforeSend  []windows.AccessibleNode
	nodesAfterSend   []windows.AccessibleNode

	// Operation tracking
	focusCalled      bool
	sendCalled       bool
	verifyCalled     bool
	lastFocusHandle  uintptr
	lastSendContent  string
	sendKeysCalls    []string
	clipboardContent string

	// Screenshot tracking
	beforeScreenshot []byte
	afterScreenshot  []byte
}

func newStateChangingMockBridge() *stateChangingMockBridge {
	return &stateChangingMockBridge{
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
		// Default nodes before focus (conversation list)
		nodesBeforeFocus: []windows.AccessibleNode{
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
		},
		// Nodes after focus (same list, but with active state)
		nodesAfterFocus: []windows.AccessibleNode{
			{
				Handle: 1,
				Name:   "张三",
				Role:   "list item selected",
				Bounds: [4]int{10, 50, 180, 40},
			},
			{
				Handle: 2,
				Name:   "李四",
				Role:   "list item",
				Bounds: [4]int{10, 90, 180, 40},
			},
			// Message area nodes
			{
				Handle: 100,
				Name:   "Message Area",
				Role:   "text",
				Bounds: [4]int{200, 100, 300, 200},
			},
		},
		// Nodes before send (no new message)
		nodesBeforeSend: []windows.AccessibleNode{
			{
				Handle: 1,
				Name:   "张三",
				Role:   "list item selected",
				Bounds: [4]int{10, 50, 180, 40},
			},
			{
				Handle: 100,
				Name:   "Message Area",
				Role:   "text",
				Bounds: [4]int{200, 100, 300, 200},
			},
		},
		// Nodes after send (with new message)
		nodesAfterSend: []windows.AccessibleNode{
			{
				Handle: 1,
				Name:   "张三",
				Role:   "list item selected",
				Bounds: [4]int{10, 50, 180, 40},
			},
			{
				Handle: 100,
				Name:   "Message Area",
				Role:   "text",
				Bounds: [4]int{200, 100, 300, 200},
			},
			{
				Handle: 101,
				Name:   "Test message",
				Role:   "text",
				Bounds: [4]int{210, 110, 280, 30},
			},
		},
		// Screenshots (simple byte arrays for testing)
		beforeScreenshot: []byte{0x01, 0x02, 0x03},
		afterScreenshot:  []byte{0x01, 0x02, 0x04}, // Different last byte
	}
}

func (m *stateChangingMockBridge) Initialize() adapter.Result {
	m.initialized = true
	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *stateChangingMockBridge) FindTopLevelWindows(className, windowName string) ([]uintptr, adapter.Result) {
	if m.findError.Status != adapter.StatusSuccess {
		return nil, m.findError
	}
	return m.findResult, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *stateChangingMockBridge) FindWindow(className, windowName string) (uintptr, adapter.Result) {
	return 0, adapter.Result{
		Status:     adapter.StatusFailed,
		ReasonCode: adapter.ReasonCode("NOT_IMPLEMENTED"),
	}
}

func (m *stateChangingMockBridge) FindChildWindow(parentHandle uintptr, className, windowName string) (uintptr, adapter.Result) {
	return 0, adapter.Result{
		Status:     adapter.StatusFailed,
		ReasonCode: adapter.ReasonCode("NOT_IMPLEMENTED"),
	}
}

func (m *stateChangingMockBridge) GetWindowText(handle uintptr) (string, adapter.Result) {
	return "", adapter.Result{
		Status:     adapter.StatusFailed,
		ReasonCode: adapter.ReasonCode("NOT_IMPLEMENTED"),
	}
}

func (m *stateChangingMockBridge) GetWindowClass(handle uintptr) (string, adapter.Result) {
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

func (m *stateChangingMockBridge) GetWindowInfo(handle uintptr) (windows.WindowInfo, adapter.Result) {
	return windows.WindowInfo{
		Handle: handle,
		Class:  m.windowClass,
		Title:  m.windowTitle,
	}, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *stateChangingMockBridge) FocusWindow(handle uintptr) adapter.Result {
	m.focusCalled = true
	m.lastFocusHandle = handle
	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *stateChangingMockBridge) EnumerateAccessibleNodes(windowHandle uintptr) ([]windows.AccessibleNode, adapter.Result) {
	if m.enumerateError.Status != adapter.StatusSuccess {
		return nil, m.enumerateError
	}

	// Return different nodes based on operation state
	if m.sendCalled {
		// After send operation
		return m.nodesAfterSend, adapter.Result{
			Status:     adapter.StatusSuccess,
			ReasonCode: adapter.ReasonOK,
		}
	} else if m.focusCalled {
		// After focus operation
		return m.nodesAfterFocus, adapter.Result{
			Status:     adapter.StatusSuccess,
			ReasonCode: adapter.ReasonOK,
		}
	}
	// Before any operation
	return m.nodesBeforeFocus, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *stateChangingMockBridge) GetAccessible(windowHandle uintptr) (*windows.IAccessible, adapter.Result) {
	return nil, adapter.Result{
		Status:     adapter.StatusFailed,
		ReasonCode: adapter.ReasonCode("NOT_IMPLEMENTED"),
	}
}

func (m *stateChangingMockBridge) CaptureWindow(handle uintptr) ([]byte, adapter.Result) {
	if m.sendCalled {
		return m.afterScreenshot, adapter.Result{
			Status:     adapter.StatusSuccess,
			ReasonCode: adapter.ReasonOK,
		}
	}
	return m.beforeScreenshot, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *stateChangingMockBridge) SendKeys(handle uintptr, keys string) adapter.Result {
	m.sendCalled = true
	m.sendKeysCalls = append(m.sendKeysCalls, keys)
	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *stateChangingMockBridge) Click(handle uintptr, x, y int) adapter.Result {
	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *stateChangingMockBridge) SetClipboardText(text string) adapter.Result {
	m.clipboardContent = text
	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *stateChangingMockBridge) GetClipboardText() (string, adapter.Result) {
	return "", adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *stateChangingMockBridge) Release() {
	m.initialized = false
}

// ==================== Diagnostic Flow Test ====================

// TestDiagnosticFlow_CompleteChain tests the complete chain: Scan -> Focus -> Send -> Verify
// with state-changing mock that simulates real node changes
func TestDiagnosticFlow_CompleteChain(t *testing.T) {
	mock := newStateChangingMockBridge()
	mock.findResult = []uintptr{12345}
	wechatAdapter := NewWeChatAdapterWithBridge(mock)

	// Step 1: Scan conversations
	conversations, scanResult := wechatAdapter.Scan(protocol.AppInstanceRef{
		AppID:      "wechat",
		InstanceID: "微信",
	})

	if scanResult.Status != adapter.StatusSuccess {
		t.Fatalf("Scan should succeed, got status: %v", scanResult.Status)
	}

	if len(conversations) == 0 {
		t.Fatal("Scan should return at least one conversation")
	}

	conv := conversations[0]

	// Verify scan diagnostics
	if len(scanResult.Diagnostics) == 0 {
		t.Error("Scan should have diagnostics")
	}

	// Step 2: Focus on conversation
	focusResult := wechatAdapter.Focus(conv)

	if focusResult.Status != adapter.StatusSuccess {
		t.Fatalf("Focus should succeed, got status: %v", focusResult.Status)
	}

	if !mock.focusCalled {
		t.Error("Focus should have called bridge.FocusWindow")
	}

	// Verify focus diagnostics
	if len(focusResult.Diagnostics) == 0 {
		t.Error("Focus should have diagnostics")
	}

	focusDiag := focusResult.Diagnostics[0].Context

	// Assert focus diagnostics match rule object fields
	if _, ok := focusDiag["locate_source"]; !ok {
		t.Error("Focus diagnostics should have locate_source field")
	}
	if _, ok := focusDiag["evidence_count"]; !ok {
		t.Error("Focus diagnostics should have evidence_count field")
	}
	if _, ok := focusDiag["confidence"]; !ok {
		t.Error("Focus diagnostics should have confidence field")
	}

	// Step 3: Send message
	sendResult := wechatAdapter.Send(conv, "Test message", "task-123")

	if sendResult.Status != adapter.StatusSuccess {
		t.Fatalf("Send should succeed, got status: %v", sendResult.Status)
	}

	if !mock.sendCalled {
		t.Error("Send should have called bridge.SendKeys")
	}

	// Verify send diagnostics
	if len(sendResult.Diagnostics) == 0 {
		t.Error("Send should have diagnostics")
	}

	sendDiag := sendResult.Diagnostics[0].Context

	// Assert send diagnostics match rule object fields
	if _, ok := sendDiag["new_message_nodes"]; !ok {
		t.Error("Send diagnostics should have new_message_nodes field")
	}
	if _, ok := sendDiag["message_content_match"]; !ok {
		t.Error("Send diagnostics should have message_content_match field")
	}
	if _, ok := sendDiag["delivery_state"]; !ok {
		t.Error("Send diagnostics should have delivery_state field")
	}
	if _, ok := sendDiag["confidence"]; !ok {
		t.Error("Send diagnostics should have confidence field")
	}

	// Step 4: Verify message delivery
	msg, verifyResult := wechatAdapter.Verify(conv, "Test message", 5*time.Second)

	if verifyResult.Status != adapter.StatusSuccess {
		t.Fatalf("Verify should succeed, got status: %v", verifyResult.Status)
	}

	// Note: In stub implementation, msg will be nil
	if msg != nil {
		t.Log("Verify returned a message (stub implementation may vary)")
	}

	// Verify verify diagnostics
	if len(verifyResult.Diagnostics) == 0 {
		t.Error("Verify should have diagnostics")
	}

	verifyDiag := verifyResult.Diagnostics[0].Context

	// Assert verify diagnostics match rule object fields
	if _, ok := verifyDiag["delivery_state"]; !ok {
		t.Error("Verify diagnostics should have delivery_state field")
	}
	if _, ok := verifyDiag["confidence"]; !ok {
		t.Error("Verify diagnostics should have confidence field")
	}
}

// TestDiagnosticFlow_StateChanges tests that the mock correctly tracks state changes
func TestDiagnosticFlow_StateChanges(t *testing.T) {
	mock := newStateChangingMockBridge()
	mock.findResult = []uintptr{12345}

	// Verify initial state
	nodes, _ := mock.EnumerateAccessibleNodes(12345)
	if len(nodes) != 2 {
		t.Errorf("Expected 2 initial nodes, got %d", len(nodes))
	}

	// Simulate focus operation
	mock.focusCalled = true
	nodes, _ = mock.EnumerateAccessibleNodes(12345)
	if len(nodes) != 3 {
		t.Errorf("Expected 3 nodes after focus, got %d", len(nodes))
	}

	// Verify active state is set
	hasActiveState := false
	for _, node := range nodes {
		if node.Role == "list item selected" {
			hasActiveState = true
			break
		}
	}
	if !hasActiveState {
		t.Error("Expected at least one node with 'selected' role after focus")
	}

	// Simulate send operation
	mock.sendCalled = true
	nodes, _ = mock.EnumerateAccessibleNodes(12345)
	if len(nodes) != 3 {
		t.Errorf("Expected 3 nodes after send, got %d", len(nodes))
	}

	// Verify new message node exists
	hasNewMessage := false
	for _, node := range nodes {
		if node.Name == "Test message" {
			hasNewMessage = true
			break
		}
	}
	if !hasNewMessage {
		t.Error("Expected 'Test message' node after send")
	}
}

// TestDiagnosticFlow_DiagnosticsConsistency tests that diagnostics are consistent across operations
func TestDiagnosticFlow_DiagnosticsConsistency(t *testing.T) {
	mock := newStateChangingMockBridge()
	mock.findResult = []uintptr{12345}
	wechatAdapter := NewWeChatAdapterWithBridge(mock)

	// Run complete chain
	conversations, _ := wechatAdapter.Scan(protocol.AppInstanceRef{
		AppID:      "wechat",
		InstanceID: "微信",
	})
	conv := conversations[0]

	focusResult := wechatAdapter.Focus(conv)
	sendResult := wechatAdapter.Send(conv, "Test message", "task-123")
	_, verifyResult := wechatAdapter.Verify(conv, "Test message", 5*time.Second)

	// Verify all operations have diagnostics
	if len(focusResult.Diagnostics) == 0 {
		t.Error("Focus should have diagnostics")
	}
	if len(sendResult.Diagnostics) == 0 {
		t.Error("Send should have diagnostics")
	}
	if len(verifyResult.Diagnostics) == 0 {
		t.Error("Verify should have diagnostics")
	}

	// Verify confidence format (2 decimal places)
	focusDiag := focusResult.Diagnostics[0].Context
	if confidence, ok := focusDiag["confidence"]; ok {
		if len(confidence) < 4 {
			t.Errorf("Confidence should have at least 4 chars (0.00), got '%s'", confidence)
		}
	}

	sendDiag := sendResult.Diagnostics[0].Context
	if confidence, ok := sendDiag["confidence"]; ok {
		if len(confidence) < 4 {
			t.Errorf("Confidence should have at least 4 chars (0.00), got '%s'", confidence)
		}
	}

	// Verify delivery state is one of the expected values
	if deliveryState, ok := sendDiag["delivery_state"]; ok {
		validStates := []string{"verified", "sent_unverified", "unknown", "failed"}
		found := false
		for _, state := range validStates {
			if deliveryState == state {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Invalid delivery_state: %s", deliveryState)
		}
	}
}

// TestDiagnosticFlow_SendKeysVerification tests that SendKeys is called correctly
func TestDiagnosticFlow_SendKeysVerification(t *testing.T) {
	mock := newStateChangingMockBridge()
	mock.findResult = []uintptr{12345}
	wechatAdapter := NewWeChatAdapterWithBridge(mock)

	conversations, _ := wechatAdapter.Scan(protocol.AppInstanceRef{
		AppID:      "wechat",
		InstanceID: "微信",
	})
	conv := conversations[0]

	wechatAdapter.Send(conv, "Test message", "task-123")

	// Verify SendKeys was called with paste and enter commands
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

	// Verify clipboard content
	if mock.clipboardContent != "Test message" {
		t.Errorf("Clipboard content mismatch: expected 'Test message', got '%s'", mock.clipboardContent)
	}
}

// TestDiagnosticFlow_ConversionFunctions tests the diagnostic conversion functions
func TestDiagnosticFlow_ConversionFunctions(t *testing.T) {
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

	_ = evidence // Use evidence variable
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
	if messageDiagnostics["new_message_nodes"] != "1" {
		t.Errorf("new_message_nodes mismatch: expected '1', got '%s'", messageDiagnostics["new_message_nodes"])
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
