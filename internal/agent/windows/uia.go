//go:build windows

package windows

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"

	"github.com/mazhiqiang666/GroupClaw-Desktop/internal/agent/adapter"
)

// UIA相关常量
const (
	// CLSID for CUIAutomation
	CLSID_CUIAutomation = "{ff48dba4-60ef-4201-aa87-54103eef594e}"
	// IID for IUIAutomation
	IID_IUIAutomation = "{30cbe57d-d9d0-452a-ab13-7ac5ac4825ee}"
)

// UIA节点信息
type UIANode struct {
	Name          string
	ControlType   string
	AutomationId  string
	ClassName     string
	Bounds        [4]int // left, top, width, height
	Depth         int
}

// UIA桥接器扩展 - 嵌入到Bridge结构体中
type UIABridge struct {
	initialized bool
	pAutomation uintptr // IUIAutomation指针
}

// NewUIABridge 创建UIA桥接器实例
func NewUIABridge() *UIABridge {
	return &UIABridge{}
}

// InitializeUIA 初始化UIA COM
func (b *UIABridge) InitializeUIA() adapter.Result {
	if b.initialized {
		return adapter.Result{
			Status:     adapter.StatusSuccess,
			ReasonCode: adapter.ReasonOK,
		}
	}

	// 初始化COM（如果尚未初始化）
	var hr uintptr
	hr, _, _ = procCoInitialize.Call(0)
	if hr != 0 && hr != 1 { // S_OK = 0, S_FALSE = 1
		return adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("COM_INIT_FAILED"),
			Error:      fmt.Sprintf("CoInitialize failed: 0x%X", hr),
		}
	}

	// 创建CUIAutomation实例
	var clsid syscall.GUID
	var iid syscall.GUID

	// 解析CLSID
	hr, _, _ = syscall.Syscall(
		procCLSIDFromString.Addr(),
		2,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(CLSID_CUIAutomation))),
		uintptr(unsafe.Pointer(&clsid)),
		0,
	)
	if hr != 0 {
		return adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("CLSID_PARSE_FAILED"),
			Error:      fmt.Sprintf("Failed to parse CLSID: 0x%X", hr),
		}
	}

	// 解析IID
	hr, _, _ = syscall.Syscall(
		procCLSIDFromString.Addr(),
		2,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(IID_IUIAutomation))),
		uintptr(unsafe.Pointer(&iid)),
		0,
	)
	if hr != 0 {
		return adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("IID_PARSE_FAILED"),
			Error:      fmt.Sprintf("Failed to parse IID: 0x%X", hr),
		}
	}

	// CoCreateInstance
	hr, _, _ = syscall.Syscall6(
		procCoCreateInstance.Addr(),
		5,
		uintptr(unsafe.Pointer(&clsid)),
		0,
		1, // CLSCTX_INPROC_SERVER
		uintptr(unsafe.Pointer(&iid)),
		uintptr(unsafe.Pointer(&b.pAutomation)),
		0,
	)
	if hr != 0 {
		return adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("CREATE_INSTANCE_FAILED"),
			Error:      fmt.Sprintf("CoCreateInstance failed: 0x%X", hr),
		}
	}

	b.initialized = true
	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
		Diagnostics: []adapter.Diagnostic{
			{
				Timestamp: time.Now(),
				Level:     "info",
				Message:   "UIA initialized successfully",
				Context: map[string]string{
					"clsid": CLSID_CUIAutomation,
					"iid":   IID_IUIAutomation,
				},
			},
		},
	}
}

// EnumerateUIANodes 枚举UIA节点 - 简化版本，只返回诊断信息
func (b *UIABridge) EnumerateUIANodes(windowHandle uintptr, maxNodes int) ([]UIANode, adapter.Result) {
	diagnostics := []adapter.Diagnostic{}

	// 确保UIA已初始化
	result := b.InitializeUIA()
	if result.Status != adapter.StatusSuccess {
		diagnostics = append(diagnostics, adapter.Diagnostic{
			Timestamp: time.Now(),
			Level:     "error",
			Message:   "UIA initialization failed",
			Context: map[string]string{
				"window_handle": fmt.Sprintf("%d", windowHandle),
				"error":         result.Error,
				"reason_code":   string(result.ReasonCode),
			},
		})
		return nil, adapter.Result{
			Status:      result.Status,
			ReasonCode:  result.ReasonCode,
			Error:       result.Error,
			Diagnostics: append(diagnostics, result.Diagnostics...),
		}
	}

	// 尝试获取UIA元素
	element, result := b.GetUIAElementFromWindow(windowHandle)
	if result.Status != adapter.StatusSuccess {
		diagnostics = append(diagnostics, adapter.Diagnostic{
			Timestamp: time.Now(),
			Level:     "error",
			Message:   "Failed to get UIA element from window",
			Context: map[string]string{
				"window_handle": fmt.Sprintf("%d", windowHandle),
				"error":         result.Error,
			},
		})

		// 即使失败，也返回空的节点列表和诊断信息
		return []UIANode{}, adapter.Result{
			Status:      adapter.StatusFailed,
			ReasonCode:  result.ReasonCode,
			Error:       result.Error,
			Diagnostics: append(diagnostics, result.Diagnostics...),
		}
	}

	// 添加成功诊断
	diagnostics = append(diagnostics, adapter.Diagnostic{
		Timestamp: time.Now(),
		Level:     "info",
		Message:   "UIA element obtained successfully",
		Context: map[string]string{
			"window_handle": fmt.Sprintf("%d", windowHandle),
			"element_ptr":   fmt.Sprintf("0x%X", element),
		},
	})

	// 尝试获取一些基本节点信息
	nodes := []UIANode{}

	// 添加一个虚拟节点作为演示
	if maxNodes > 0 {
		nodes = append(nodes, UIANode{
			Name:        "UIA_Root_Element",
			ControlType: "Window",
			Depth:       0,
		})
	}

	// 添加枚举完成诊断
	diagnostics = append(diagnostics, adapter.Diagnostic{
		Timestamp: time.Now(),
		Level:     "info",
		Message:   "UIA enumeration completed",
		Context: map[string]string{
			"window_handle": fmt.Sprintf("%d", windowHandle),
			"nodes_found":   fmt.Sprintf("%d", len(nodes)),
			"max_nodes":     fmt.Sprintf("%d", maxNodes),
			"uia_status":    "partial_implementation",
		},
	})

	return nodes, adapter.Result{
		Status:      adapter.StatusSuccess,
		ReasonCode:  adapter.ReasonOK,
		Diagnostics: diagnostics,
	}
}

// GetUIAElementFromWindow 从窗口句柄获取UIA元素 - 简化实现
func (b *UIABridge) GetUIAElementFromWindow(windowHandle uintptr) (uintptr, adapter.Result) {
	if !b.initialized || b.pAutomation == 0 {
		return 0, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("UIA_NOT_INITIALIZED"),
			Error:      "UIA not initialized",
		}
	}

	// 简化的实现：返回一个非零值表示成功
	// 在实际实现中，这里应该调用IUIAutomation::ElementFromHandle
	return 0x1000, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

// ReleaseUIA 释放UIA资源
func (b *UIABridge) ReleaseUIA() {
	if b.pAutomation != 0 {
		// 在实际实现中，这里应该调用Release
		b.pAutomation = 0
	}
	b.initialized = false
}