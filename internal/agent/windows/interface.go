package windows

import (
	"github.com/mazhiqiang666/GroupClaw-Desktop/internal/agent/adapter"
)

// InputBoxRect 输入框矩形和特征
type InputBoxRect struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// InputBoxCandidate 输入框候选区域
type InputBoxCandidate struct {
	Index        int               `json:"index"`        // 候选索引
	Rect         InputBoxRect      `json:"rect"`         // 矩形区域
	Source       string            `json:"source"`       // 来源：visual/geometric/ocr
	Score        int               `json:"score"`        // 综合评分
	Features     map[string]string `json:"features"`     // 特征描述
	ActivationScore float64         `json:"activation_score"` // 激活评分
	ActivationSignals []string      `json:"activation_signals"` // 激活信号
	EditableConfidence float64      `json:"editable_confidence"` // 可编辑控件置信度
	RejectedReason string           `json:"rejected_reason"`    // 拒绝原因
}

// InputBoxProbeResult 候选激活验证结果
type InputBoxProbeResult struct {
	CandidateIndex     int               `json:"candidate_index"`
	ActivationScore    float64           `json:"activation_score"`
	ActivationSignals  []string          `json:"activation_signals"`
	WeakSignals        []string          `json:"weak_signals"`      // 弱信号（位置稳定等）
	StrongSignals      []string          `json:"strong_signals"`    // 强信号（视觉变化、焦点变化等）
	EditableConfidence float64           `json:"editable_confidence"`
	RejectedReason     string            `json:"rejected_reason"`
	BeforeImage        []byte            `json:"-"` // 不序列化
	AfterImage         []byte            `json:"-"` // 不序列化
	DebugImagePath     string            `json:"debug_image_path"`
}

// BridgeInterface 定义 Windows UIA 桥接器接口
// 为 WeChat adapter 提供最小可调用的 Windows 操作接口
type BridgeInterface interface {
	// Initialize 初始化 COM
	Initialize() adapter.Result

	// FindTopLevelWindows 查找顶级窗口（按类名或标题）
	FindTopLevelWindows(className, windowName string) ([]uintptr, adapter.Result)

	// FindWindow 查找单个窗口
	FindWindow(className, windowName string) (uintptr, adapter.Result)

	// FindChildWindow 查找子窗口
	FindChildWindow(parentHandle uintptr, className, windowName string) (uintptr, adapter.Result)

	// GetWindowText 获取窗口标题
	GetWindowText(handle uintptr) (string, adapter.Result)

	// GetWindowClass 获取窗口类名
	GetWindowClass(handle uintptr) (string, adapter.Result)

	// GetWindowInfo 获取窗口信息
	GetWindowInfo(handle uintptr) (WindowInfo, adapter.Result)

	// FocusWindow 聚焦到窗口
	FocusWindow(handle uintptr) adapter.Result

	// EnumerateAccessibleNodes 枚举可访问节点（用于 UIA 遍历）
	EnumerateAccessibleNodes(windowHandle uintptr) ([]AccessibleNode, adapter.Result)

	// GetAccessible 获取可访问对象
	GetAccessible(windowHandle uintptr) (*IAccessible, adapter.Result)

	// CaptureWindow 截图窗口
	CaptureWindow(handle uintptr) ([]byte, adapter.Result)

	// SendKeys 发送按键
	SendKeys(handle uintptr, keys string) adapter.Result

	// Click 点击窗口位置
	Click(handle uintptr, x, y int) adapter.Result

	// SetClipboardText 设置剪贴板文本
	SetClipboardText(text string) adapter.Result

	// GetClipboardText 获取剪贴板文本
	GetClipboardText() (string, adapter.Result)

	// Release 释放资源
	Release()

	// FocusConversationByVision 视觉Focus统一入口
	FocusConversationByVision(windowHandle uintptr, strategy string, targetIndex int, waitAfterClickMs int) (VisionFocusResult, adapter.Result)

	// DetectConversations 视觉检测会话列表
	DetectConversations(windowHandle uintptr) (VisionDebugResult, adapter.Result)

	// DetectInputBoxArea 检测输入框区域（返回多候选）
	DetectInputBoxArea(windowHandle uintptr, leftSidebarRect [4]int, windowWidth, windowHeight int) ([]InputBoxCandidate, adapter.Result)

	// ProbeInputBoxCandidate 验证输入框候选激活状态
	ProbeInputBoxCandidate(windowHandle uintptr, candidate InputBoxCandidate, strategy string) (InputBoxProbeResult, adapter.Result)

	// GetInputBoxClickPoint 获取输入框点击坐标
	// strategy: 点击策略 (input_left_third, input_center, input_left_quarter, input_double_click_center)
	// 如果为空，默认为 input_left_third
	GetInputBoxClickPoint(inputBox InputBoxRect, strategy string) (x, y int, clickSource string)
}

// AccessibleNode 可访问节点信息
type AccessibleNode struct {
	Handle      uintptr
	Name        string
	Role        string
	Value       string
	ClassName   string
	Bounds      [4]int // x, y, width, height
	Children    []AccessibleNode
	TreePath    string // Hierarchical path like [0].[3].[2]
}
