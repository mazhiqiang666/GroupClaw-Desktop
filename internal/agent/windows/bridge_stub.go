//go:build !windows

package windows

import (
	"github.com/yourorg/auto-customer-service/internal/agent/adapter"
)

// 确保 Bridge 实现了 BridgeInterface 接口
var _ BridgeInterface = (*Bridge)(nil)

// WindowInfo 窗口信息
type WindowInfo struct {
	Handle uintptr
	Class  string
	Title  string
}

// Bridge Windows UIA 桥接器（非 Windows 平台 stub）
type Bridge struct{}

// NewBridge 创建桥接器实例
func NewBridge() *Bridge {
	return &Bridge{}
}

// Initialize 初始化 COM（非 Windows 平台返回不支持）
func (b *Bridge) Initialize() adapter.Result {
	return adapter.Result{
		Status:     adapter.StatusFailed,
		ReasonCode: adapter.ReasonCode("PLATFORM_NOT_SUPPORTED"),
		Error:      "Windows UIA bridge is only available on Windows",
	}
}

// FindTopLevelWindows 查找顶级窗口（非 Windows 平台返回不支持）
func (b *Bridge) FindTopLevelWindows(className, windowName string) ([]uintptr, adapter.Result) {
	return nil, adapter.Result{
		Status:     adapter.StatusFailed,
		ReasonCode: adapter.ReasonCode("PLATFORM_NOT_SUPPORTED"),
	}
}

// FindWindow 查找窗口（非 Windows 平台返回不支持）
func (b *Bridge) FindWindow(className, windowName string) (uintptr, adapter.Result) {
	return 0, adapter.Result{
		Status:     adapter.StatusFailed,
		ReasonCode: adapter.ReasonCode("PLATFORM_NOT_SUPPORTED"),
	}
}

// FindChildWindow 查找子窗口（非 Windows 平台返回不支持）
func (b *Bridge) FindChildWindow(parentHandle uintptr, className, windowName string) (uintptr, adapter.Result) {
	return 0, adapter.Result{
		Status:     adapter.StatusFailed,
		ReasonCode: adapter.ReasonCode("PLATFORM_NOT_SUPPORTED"),
	}
}

// GetWindowText 获取窗口标题（非 Windows 平台返回不支持）
func (b *Bridge) GetWindowText(handle uintptr) (string, adapter.Result) {
	return "", adapter.Result{
		Status:     adapter.StatusFailed,
		ReasonCode: adapter.ReasonCode("PLATFORM_NOT_SUPPORTED"),
	}
}

// GetWindowClass 获取窗口类名（非 Windows 平台返回不支持）
func (b *Bridge) GetWindowClass(handle uintptr) (string, adapter.Result) {
	return "", adapter.Result{
		Status:     adapter.StatusFailed,
		ReasonCode: adapter.ReasonCode("PLATFORM_NOT_SUPPORTED"),
	}
}

// GetWindowInfo 获取窗口信息（非 Windows 平台返回不支持）
func (b *Bridge) GetWindowInfo(handle uintptr) (WindowInfo, adapter.Result) {
	return WindowInfo{}, adapter.Result{
		Status:     adapter.StatusFailed,
		ReasonCode: adapter.ReasonCode("PLATFORM_NOT_SUPPORTED"),
	}
}

// FocusWindow 聚焦到窗口（非 Windows 平台返回不支持）
func (b *Bridge) FocusWindow(handle uintptr) adapter.Result {
	return adapter.Result{
		Status:     adapter.StatusFailed,
		ReasonCode: adapter.ReasonCode("PLATFORM_NOT_SUPPORTED"),
	}
}

// EnumerateAccessibleNodes 枚举可访问节点（非 Windows 平台返回不支持）
func (b *Bridge) EnumerateAccessibleNodes(windowHandle uintptr) ([]AccessibleNode, adapter.Result) {
	return nil, adapter.Result{
		Status:     adapter.StatusFailed,
		ReasonCode: adapter.ReasonCode("PLATFORM_NOT_SUPPORTED"),
	}
}

// GetAccessible 获取可访问对象（非 Windows 平台返回不支持）
func (b *Bridge) GetAccessible(windowHandle uintptr) (*IAccessible, adapter.Result) {
	return nil, adapter.Result{
		Status:     adapter.StatusFailed,
		ReasonCode: adapter.ReasonCode("PLATFORM_NOT_SUPPORTED"),
	}
}

// CaptureWindow 截图窗口（非 Windows 平台返回不支持）
func (b *Bridge) CaptureWindow(handle uintptr) ([]byte, adapter.Result) {
	return nil, adapter.Result{
		Status:     adapter.StatusFailed,
		ReasonCode: adapter.ReasonCode("PLATFORM_NOT_SUPPORTED"),
	}
}

// SendKeys 发送按键（非 Windows 平台返回不支持）
func (b *Bridge) SendKeys(handle uintptr, keys string) adapter.Result {
	return adapter.Result{
		Status:     adapter.StatusFailed,
		ReasonCode: adapter.ReasonCode("PLATFORM_NOT_SUPPORTED"),
	}
}

// Click 点击窗口位置（非 Windows 平台返回不支持）
func (b *Bridge) Click(handle uintptr, x, y int) adapter.Result {
	return adapter.Result{
		Status:     adapter.StatusFailed,
		ReasonCode: adapter.ReasonCode("PLATFORM_NOT_SUPPORTED"),
	}
}

// SetClipboardText 设置剪贴板文本（非 Windows 平台返回不支持）
func (b *Bridge) SetClipboardText(text string) adapter.Result {
	return adapter.Result{
		Status:     adapter.StatusFailed,
		ReasonCode: adapter.ReasonCode("PLATFORM_NOT_SUPPORTED"),
	}
}

// GetClipboardText 获取剪贴板文本（非 Windows 平台返回不支持）
func (b *Bridge) GetClipboardText() (string, adapter.Result) {
	return "", adapter.Result{
		Status:     adapter.StatusFailed,
		ReasonCode: adapter.ReasonCode("PLATFORM_NOT_SUPPORTED"),
	}
}

// Release 释放资源（非 Windows 平台无操作）
func (b *Bridge) Release() {}

// IAccessible COM 接口（stub）
type IAccessible struct{}
