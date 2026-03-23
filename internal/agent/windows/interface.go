package windows

import (
	"github.com/yourorg/auto-customer-service/internal/agent/adapter"
)

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
