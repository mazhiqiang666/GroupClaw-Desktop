package wechat

import (
	"time"

	"github.com/mazhiqiang666/GroupClaw-Desktop/internal/agent/adapter"
	"github.com/mazhiqiang666/GroupClaw-Desktop/internal/agent/windows"
)

// ControlledMockBridge is a mock bridge for controlled testing with full control over behavior
type ControlledMockBridge struct {
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

// NewControlledMockBridge creates a new controlled mock bridge
func NewControlledMockBridge() *ControlledMockBridge {
	return &ControlledMockBridge{
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

// SetFindResult sets the find result for testing
func (m *ControlledMockBridge) SetFindResult(result []uintptr) {
	m.findResult = result
}

// SetNodes sets the nodes for testing
func (m *ControlledMockBridge) SetNodes(nodes []windows.AccessibleNode) {
	m.nodes = nodes
}

func (m *ControlledMockBridge) Initialize() adapter.Result {
	m.initialized = true
	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *ControlledMockBridge) FindTopLevelWindows(className, windowName string) ([]uintptr, adapter.Result) {
	if m.findError.Status != adapter.StatusSuccess {
		return nil, m.findError
	}
	return m.findResult, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *ControlledMockBridge) FindWindow(className, windowName string) (uintptr, adapter.Result) {
	return 0, adapter.Result{
		Status:     adapter.StatusFailed,
		ReasonCode: adapter.ReasonCode("NOT_IMPLEMENTED"),
	}
}

func (m *ControlledMockBridge) FindChildWindow(parentHandle uintptr, className, windowName string) (uintptr, adapter.Result) {
	return 0, adapter.Result{
		Status:     adapter.StatusFailed,
		ReasonCode: adapter.ReasonCode("NOT_IMPLEMENTED"),
	}
}

func (m *ControlledMockBridge) GetWindowText(handle uintptr) (string, adapter.Result) {
	return "", adapter.Result{
		Status:     adapter.StatusFailed,
		ReasonCode: adapter.ReasonCode("NOT_IMPLEMENTED"),
	}
}

func (m *ControlledMockBridge) GetWindowClass(handle uintptr) (string, adapter.Result) {
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

func (m *ControlledMockBridge) GetWindowInfo(handle uintptr) (windows.WindowInfo, adapter.Result) {
	return windows.WindowInfo{
		Handle: handle,
		Class:  m.windowClass,
		Title:  m.windowTitle,
	}, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *ControlledMockBridge) FocusWindow(handle uintptr) adapter.Result {
	m.focusCalled = true
	m.lastFocusHandle = handle
	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *ControlledMockBridge) EnumerateAccessibleNodes(windowHandle uintptr) ([]windows.AccessibleNode, adapter.Result) {
	if m.enumerateError.Status != adapter.StatusSuccess {
		return nil, m.enumerateError
	}
	return m.nodes, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *ControlledMockBridge) GetAccessible(windowHandle uintptr) (*windows.IAccessible, adapter.Result) {
	return nil, adapter.Result{
		Status:     adapter.StatusFailed,
		ReasonCode: adapter.ReasonCode("NOT_IMPLEMENTED"),
	}
}

func (m *ControlledMockBridge) CaptureWindow(handle uintptr) ([]byte, adapter.Result) {
	return []byte{}, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *ControlledMockBridge) SendKeys(handle uintptr, keys string) adapter.Result {
	m.sendCalled = true
	m.lastSendContent = keys
	m.sendKeysCalls = append(m.sendKeysCalls, keys)
	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *ControlledMockBridge) Click(handle uintptr, x, y int) adapter.Result {
	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *ControlledMockBridge) SetClipboardText(text string) adapter.Result {
	m.clipboardContent = text
	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *ControlledMockBridge) GetClipboardText() (string, adapter.Result) {
	return "", adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *ControlledMockBridge) Release() {
	m.initialized = false
}

// StateChangingMockBridge is a mock bridge that tracks node state changes
// based on actual operations (click, send keys) to simulate real-world behavior
type StateChangingMockBridge struct {
	initialized    bool
	findResult     []uintptr
	findError      adapter.Result
	windowClass    string
	windowTitle    string
	enumerateError adapter.Result

	// Session state tracking
	currentActiveSession uintptr // Handle of currently active session
	sendState            sendStateType

	// Node definitions for different states
	nodesInitial      []windows.AccessibleNode // Initial conversation list
	nodesAfterFocus   []windows.AccessibleNode // After focusing on a session
	nodesAfterSend    []windows.AccessibleNode // After sending a message

	// Operation tracking
	lastFocusHandle  uintptr
	lastSendContent  string
	sendKeysCalls    []string
	clipboardContent string

	// Screenshot tracking
	beforeScreenshot []byte
	afterScreenshot  []byte
}

// sendStateType tracks the progress of send operation
type sendStateType int

const (
	sendStateNone sendStateType = iota
	sendStatePasteCalled
	sendStateEnterCalled
)

// NewStateChangingMockBridge creates a new state-changing mock bridge
func NewStateChangingMockBridge() *StateChangingMockBridge {
	return &StateChangingMockBridge{
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
		// Initial conversation list (no session selected)
		nodesInitial: []windows.AccessibleNode{
			{
				Handle:    1,
				Name:      "张三",
				Role:      "list item",
				ClassName: "ListViewItem",
				Bounds:    [4]int{10, 50, 180, 40},
				TreePath:  "[0]",
				Children:  []windows.AccessibleNode{},
			},
			{
				Handle:    2,
				Name:      "李四",
				Role:      "list item",
				ClassName: "ListViewItem",
				Bounds:    [4]int{10, 90, 180, 40},
				TreePath:  "[1]",
				Children:  []windows.AccessibleNode{},
			},
		},
		// Nodes after focus (session selected, message area visible)
		nodesAfterFocus: []windows.AccessibleNode{
			{
				Handle:    1,
				Name:      "张三",
				Role:      "list item selected",
				ClassName: "ListViewItem",
				Bounds:    [4]int{10, 50, 180, 40},
				TreePath:  "[0]",
				Children:  []windows.AccessibleNode{},
			},
			{
				Handle:    2,
				Name:      "李四",
				Role:      "list item",
				ClassName: "ListViewItem",
				Bounds:    [4]int{10, 90, 180, 40},
				TreePath:  "[1]",
				Children:  []windows.AccessibleNode{},
			},
			// Message area nodes
			{
				Handle:    100,
				Name:      "Message Area",
				Role:      "text",
				ClassName: "Edit",
				Bounds:    [4]int{200, 100, 300, 200},
				TreePath:  "[2]",
				Children:  []windows.AccessibleNode{},
			},
		},
		// Nodes after send (with new message node)
		nodesAfterSend: []windows.AccessibleNode{
			{
				Handle:    1,
				Name:      "张三",
				Role:      "list item selected",
				ClassName: "ListViewItem",
				Bounds:    [4]int{10, 50, 180, 40},
				TreePath:  "[0]",
				Children:  []windows.AccessibleNode{},
			},
			{
				Handle:    2,
				Name:      "李四",
				Role:      "list item",
				ClassName: "ListViewItem",
				Bounds:    [4]int{10, 90, 180, 40},
				TreePath:  "[1]",
				Children:  []windows.AccessibleNode{},
			},
			{
				Handle:    100,
				Name:      "Message Area",
				Role:      "text",
				ClassName: "Edit",
				Bounds:    [4]int{200, 100, 300, 200},
				TreePath:  "[2]",
				Children:  []windows.AccessibleNode{},
			},
			{
				Handle:    101,
				Name:      "Test message",
				Role:      "text",
				ClassName: "Static",
				Bounds:    [4]int{210, 110, 280, 30},
				TreePath:  "[2].[0]",
				Children:  []windows.AccessibleNode{},
			},
		},
		// Screenshots (simple byte arrays for testing)
		beforeScreenshot: []byte{0x01, 0x02, 0x03},
		afterScreenshot:  []byte{0x01, 0x02, 0x04}, // Different last byte
	}
}

// SetFindResult sets the find result for testing
func (m *StateChangingMockBridge) SetFindResult(result []uintptr) {
	m.findResult = result
}

func (m *StateChangingMockBridge) Initialize() adapter.Result {
	m.initialized = true
	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *StateChangingMockBridge) FindTopLevelWindows(className, windowName string) ([]uintptr, adapter.Result) {
	if m.findError.Status != adapter.StatusSuccess {
		return nil, m.findError
	}
	return m.findResult, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *StateChangingMockBridge) FindWindow(className, windowName string) (uintptr, adapter.Result) {
	return 0, adapter.Result{
		Status:     adapter.StatusFailed,
		ReasonCode: adapter.ReasonCode("NOT_IMPLEMENTED"),
	}
}

func (m *StateChangingMockBridge) FindChildWindow(parentHandle uintptr, className, windowName string) (uintptr, adapter.Result) {
	return 0, adapter.Result{
		Status:     adapter.StatusFailed,
		ReasonCode: adapter.ReasonCode("NOT_IMPLEMENTED"),
	}
}

func (m *StateChangingMockBridge) GetWindowText(handle uintptr) (string, adapter.Result) {
	return "", adapter.Result{
		Status:     adapter.StatusFailed,
		ReasonCode: adapter.ReasonCode("NOT_IMPLEMENTED"),
	}
}

func (m *StateChangingMockBridge) GetWindowClass(handle uintptr) (string, adapter.Result) {
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

func (m *StateChangingMockBridge) GetWindowInfo(handle uintptr) (windows.WindowInfo, adapter.Result) {
	return windows.WindowInfo{
		Handle: handle,
		Class:  m.windowClass,
		Title:  m.windowTitle,
	}, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *StateChangingMockBridge) FocusWindow(handle uintptr) adapter.Result {
	m.lastFocusHandle = handle
	m.currentActiveSession = handle
	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *StateChangingMockBridge) EnumerateAccessibleNodes(windowHandle uintptr) ([]windows.AccessibleNode, adapter.Result) {
	if m.enumerateError.Status != adapter.StatusSuccess {
		return nil, m.enumerateError
	}

	// Return different nodes based on current state
	// State progression: initial -> after focus -> after send (paste + enter)
	if m.sendState == sendStateEnterCalled {
		// After send operation (paste + enter both called)
		return m.nodesAfterSend, adapter.Result{
			Status:     adapter.StatusSuccess,
			ReasonCode: adapter.ReasonOK,
		}
	} else if m.currentActiveSession != 0 {
		// After focus operation (session selected)
		return m.nodesAfterFocus, adapter.Result{
			Status:     adapter.StatusSuccess,
			ReasonCode: adapter.ReasonOK,
		}
	}
	// Initial state (no session selected)
	return m.nodesInitial, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *StateChangingMockBridge) GetAccessible(windowHandle uintptr) (*windows.IAccessible, adapter.Result) {
	return nil, adapter.Result{
		Status:     adapter.StatusFailed,
		ReasonCode: adapter.ReasonCode("NOT_IMPLEMENTED"),
	}
}

func (m *StateChangingMockBridge) CaptureWindow(handle uintptr) ([]byte, adapter.Result) {
	if m.sendState == sendStateEnterCalled {
		// After send operation - screenshot should show new message
		return m.afterScreenshot, adapter.Result{
			Status:     adapter.StatusSuccess,
			ReasonCode: adapter.ReasonOK,
		}
	}
	// Before send operation
	return m.beforeScreenshot, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *StateChangingMockBridge) SendKeys(handle uintptr, keys string) adapter.Result {
	m.sendKeysCalls = append(m.sendKeysCalls, keys)

	// Track send state progression
	if keys == "^v" { // Paste command
		m.sendState = sendStatePasteCalled
	} else if keys == "{ENTER}" { // Enter command
		m.sendState = sendStateEnterCalled
	}

	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *StateChangingMockBridge) Click(handle uintptr, x, y int) adapter.Result {
	// Simulate clicking on a conversation item in the list
	// Check if click is within the conversation list area (left side of window)
	if x < 200 { // Conversation list is on the left side
		// Find which conversation was clicked based on Y coordinate
		for _, node := range m.nodesInitial {
			if y >= node.Bounds[1] && y <= node.Bounds[1]+node.Bounds[3] {
				m.currentActiveSession = node.Handle
				m.lastFocusHandle = node.Handle
				break
			}
		}
	}
	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *StateChangingMockBridge) SetClipboardText(text string) adapter.Result {
	m.clipboardContent = text
	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *StateChangingMockBridge) GetClipboardText() (string, adapter.Result) {
	return "", adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

func (m *StateChangingMockBridge) FocusConversationByVision(windowHandle uintptr, strategy string, targetIndex int, waitAfterClickMs int) (windows.VisionFocusResult, adapter.Result) {
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

func (m *StateChangingMockBridge) DetectConversations(windowHandle uintptr) (windows.VisionDebugResult, adapter.Result) {
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

func (m *StateChangingMockBridge) Release() {
	m.initialized = false
}

// DetectInputBoxArea 检测输入框区域（mock实现）
func (m *StateChangingMockBridge) DetectInputBoxArea(windowHandle uintptr, leftSidebarRect [4]int, windowWidth, windowHeight int) (windows.InputBoxRect, adapter.Result) {
	// 返回模拟的输入框矩形
	rect := windows.InputBoxRect{
		X:      leftSidebarRect[0] + leftSidebarRect[2] + 20,
		Y:      windowHeight - 100,
		Width:  windowWidth - leftSidebarRect[2] - 40,
		Height: 80,
	}
	return rect, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

// GetInputBoxClickPoint 获取输入框点击坐标（mock实现）
func (m *StateChangingMockBridge) GetInputBoxClickPoint(inputBox windows.InputBoxRect) (x, y int, clickSource string) {
	// 点击输入框左侧1/3处，垂直居中
	x = inputBox.X + inputBox.Width/3
	y = inputBox.Y + inputBox.Height/2
	return x, y, "input_box_left_third_mock"
}
