//go:build windows

package windows

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"

	"github.com/mazhiqiang666/GroupClaw-Desktop/internal/agent/adapter"
)

var (
	moduiautomation = syscall.NewLazyDLL("uiautomation.dll")

	// IUIAutomation vtable methods (to be called via COM)
	// 注意：这些不是DLL导出函数，而是通过COM接口调用
)

// UIA相关常量
const (
	// CLSID for CUIAutomation
	CLSID_CUIAutomation = "{ff48dba4-60ef-4201-aa87-54103eef594e}"
	// IID for IUIAutomation
	IID_IUIAutomation = "{30cbe57d-d9d0-452a-ab13-7ac5ac4825ee}"

	// TreeScope constants
	TreeScope_Element     = 0x1
	TreeScope_Children    = 0x2
	TreeScope_Descendants = 0x4
	TreeScope_Parent      = 0x8
	TreeScope_Ancestors   = 0x10
	TreeScope_Subtree     = TreeScope_Element | TreeScope_Children | TreeScope_Descendants
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

// IUIAutomation COM接口
type IUIAutomation struct {
	lpVtbl *uintptr
}

// IUIAutomationVtbl 虚函数表（简化版，只包含我们需要的方法）
type IUIAutomationVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	// IUIAutomation方法
	CompareElements        uintptr
	CompareRuntimeIds      uintptr
	GetRootElement         uintptr
	ElementFromHandle      uintptr
	ElementFromPoint       uintptr
	GetFocusedElement      uintptr
	GetRootElementBuildCache uintptr
	ElementFromHandleBuildCache uintptr
	ElementFromPointBuildCache uintptr
	GetFocusedElementBuildCache uintptr
	CreateTreeWalker       uintptr
	CreateCacheRequest     uintptr
	CreateTrueCondition    uintptr
	CreateFalseCondition   uintptr
	CreatePropertyCondition uintptr
	CreateAndCondition     uintptr
	CreateOrCondition      uintptr
	CreateNotCondition     uintptr
	AddAutomationEventHandler uintptr
	RemoveAutomationEventHandler uintptr
	AddPropertyChangedEventHandler uintptr
	RemovePropertyChangedEventHandler uintptr
	AddStructureChangedEventHandler uintptr
	RemoveStructureChangedEventHandler uintptr
	AddFocusChangedEventHandler uintptr
	RemoveFocusChangedEventHandler uintptr
	RemoveAllEventHandlers uintptr
	IntNativeArrayToSafeArray uintptr
	IntSafeArrayToNativeArray uintptr
	RectToVariant          uintptr
	VariantToRect          uintptr
	SafeArrayToRectNativeArray uintptr
	CreateProxyFactoryEntry uintptr
	GetPropertyProgrammaticName uintptr
	GetPatternProgrammaticName uintptr
	PollForPotentialSupportedPatterns uintptr
	PollForPotentialSupportedProperties uintptr
	CheckNotSupported      uintptr
	GetReservedNotSupportedValue uintptr
	GetReservedMixedAttributeValue uintptr
	ElementFromIAccessible uintptr
	ElementFromIAccessibleBuildCache uintptr
}

// IUIAutomationElement COM接口
type IUIAutomationElement struct {
	lpVtbl *uintptr
}

// IUIAutomationElementVtbl 虚函数表（简化版）
type IUIAutomationElementVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	// IUIAutomationElement方法
	SetFocus                uintptr
	GetRuntimeId            uintptr
	FindFirst               uintptr
	FindAll                 uintptr
	GetCurrentPropertyValue uintptr
	GetCurrentPropertyValueEx uintptr
	GetCachedPropertyValue  uintptr
	GetCachedPropertyValueEx uintptr
	GetCurrentPatternAs     uintptr
	GetCachedPatternAs      uintptr
	GetCurrentPatternObject uintptr
	GetCachedPatternObject  uintptr
	GetCachedParent         uintptr
	GetCachedChildren       uintptr
	GetCurrentControlType   uintptr
	GetCurrentLocalizedControlType uintptr
	GetCurrentName          uintptr
	GetCurrentAcceleratorKey uintptr
	GetCurrentAccessKey     uintptr
	GetCurrentHasKeyboardFocus uintptr
	GetCurrentIsKeyboardFocusable uintptr
	GetCurrentIsEnabled     uintptr
	GetCurrentAutomationId   uintptr
	GetCurrentClassName     uintptr
	GetCurrentHelpText      uintptr
	GetCurrentCulture       uintptr
	GetCurrentIsControlElement uintptr
	GetCurrentIsContentElement uintptr
	GetCurrentLabeledBy     uintptr
	GetCurrentAriaRole      uintptr
	GetCurrentAriaProperties uintptr
	GetCurrentIsPassword    uintptr
	GetCurrentNativeWindowHandle uintptr
	GetCurrentItemType      uintptr
	GetCurrentIsOffscreen   uintptr
	GetCurrentOrientation   uintptr
	GetCurrentFrameworkId   uintptr
	GetCurrentIsRequiredForForm uintptr
	GetCurrentItemStatus    uintptr
	GetCurrentBoundingRectangle uintptr
	GetCurrentClickablePoint uintptr
}

// UIA桥接器扩展 - 嵌入到Bridge结构体中
type UIABridge struct {
	initialized bool
	pAutomation *IUIAutomation // IUIAutomation指针
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
			"element_ptr":   fmt.Sprintf("0x%X", uintptr(unsafe.Pointer(element))),
			"uia_init_ok":   "true",
			"element_from_handle_ok": "true",
		},
	})

	// 尝试获取一些基本节点信息
	nodes := []UIANode{}
	realNodesFound := 0
	placeholderMode := "false"

	// 尝试获取根元素的名称
	name, nameResult := b.getElementName(element)
	nameInfo := "unknown"
	if nameResult.Status == adapter.StatusSuccess {
		nameInfo = name
		if nameInfo == "" {
			nameInfo = "(empty)"
		}
	} else {
		nameInfo = fmt.Sprintf("error: %s", nameResult.Error)
	}

	// 添加根节点信息
	if maxNodes > 0 {
		nodes = append(nodes, UIANode{
			Name:        fmt.Sprintf("UIA_Root: %s", nameInfo),
			ControlType: "Window", // 假设是窗口
			Depth:       0,
		})
		realNodesFound++
	}

	// 尝试获取更多诊断信息
	diagnostics = append(diagnostics, adapter.Diagnostic{
		Timestamp: time.Now(),
		Level:     "info",
		Message:   "UIA root element analyzed",
		Context: map[string]string{
			"root_name":      nameInfo,
			"real_nodes_found": fmt.Sprintf("%d", realNodesFound),
			"placeholder_mode": placeholderMode,
		},
	})

	// 添加枚举完成诊断
	diagnostics = append(diagnostics, adapter.Diagnostic{
		Timestamp: time.Now(),
		Level:     "info",
		Message:   "UIA enumeration completed",
		Context: map[string]string{
			"window_handle":     fmt.Sprintf("%d", windowHandle),
			"nodes_found":       fmt.Sprintf("%d", len(nodes)),
			"max_nodes":         fmt.Sprintf("%d", maxNodes),
			"uia_status":        "real_implementation_basic",
			"real_nodes_found":  fmt.Sprintf("%d", realNodesFound),
			"placeholder_mode":  placeholderMode,
			"uia_init_ok":       "true",
			"element_from_handle_ok": "true",
			"tree_walk_ok":      "false", // 尚未实现TreeWalker
			"findall_ok":        "false", // 尚未实现FindAll
		},
	})

	return nodes, adapter.Result{
		Status:      adapter.StatusSuccess,
		ReasonCode:  adapter.ReasonOK,
		Diagnostics: diagnostics,
	}
}

// GetUIAElementFromWindow 从窗口句柄获取UIA元素 - 真实实现
func (b *UIABridge) GetUIAElementFromWindow(windowHandle uintptr) (*IUIAutomationElement, adapter.Result) {
	if !b.initialized || b.pAutomation == nil {
		return nil, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("UIA_NOT_INITIALIZED"),
			Error:      "UIA not initialized",
		}
	}

	// 获取IUIAutomation的vtable
	vtbl := (*IUIAutomationVtbl)(unsafe.Pointer(b.pAutomation.lpVtbl))
	if vtbl.ElementFromHandle == 0 {
		return nil, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("UIA_METHOD_NOT_FOUND"),
			Error:      "ElementFromHandle method not found in vtable",
		}
	}

	// 调用ElementFromHandle
	var element *IUIAutomationElement
	hr, _, _ := syscall.Syscall(
		vtbl.ElementFromHandle,
		3,
		uintptr(unsafe.Pointer(b.pAutomation)),
		windowHandle,
		uintptr(unsafe.Pointer(&element)),
	)

	if hr != 0 {
		return nil, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("ELEMENT_FROM_HANDLE_FAILED"),
			Error:      fmt.Sprintf("ElementFromHandle failed: 0x%X", hr),
		}
	}

	if element == nil {
		return nil, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("ELEMENT_IS_NULL"),
			Error:      "ElementFromHandle returned null element",
		}
	}

	return element, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

// getElementName 获取UIA元素的名称
func (b *UIABridge) getElementName(element *IUIAutomationElement) (string, adapter.Result) {
	if element == nil {
		return "", adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("ELEMENT_IS_NULL"),
			Error:      "Element is null",
		}
	}

	vtbl := (*IUIAutomationElementVtbl)(unsafe.Pointer(element.lpVtbl))
	if vtbl.GetCurrentName == 0 {
		return "", adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("UIA_METHOD_NOT_FOUND"),
			Error:      "GetCurrentName method not found in vtable",
		}
	}

	var namePtr *uint16
	hr, _, _ := syscall.Syscall(
		vtbl.GetCurrentName,
		2,
		uintptr(unsafe.Pointer(element)),
		uintptr(unsafe.Pointer(&namePtr)),
		0,
	)

	if hr != 0 {
		return "", adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("GET_NAME_FAILED"),
			Error:      fmt.Sprintf("GetCurrentName failed: 0x%X", hr),
		}
	}

	if namePtr == nil {
		return "", adapter.Result{}
	}

	name := syscall.UTF16ToString((*[1 << 20]uint16)(unsafe.Pointer(namePtr))[:])
	// 释放BSTR内存
	modole32 := syscall.NewLazyDLL("ole32.dll")
	procSysFreeString := modole32.NewProc("SysFreeString")
	procSysFreeString.Call(uintptr(unsafe.Pointer(namePtr)))

	return name, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

// ReleaseUIA 释放UIA资源
func (b *UIABridge) ReleaseUIA() {
	if b.pAutomation != nil {
		// 在实际实现中，这里应该调用Release
		b.pAutomation = nil
	}
	b.initialized = false
}