//go:build windows

package windows

import (
	"fmt"
	"strconv"
	"syscall"
	"time"
	"unsafe"

	"github.com/mazhiqiang666/GroupClaw-Desktop/internal/agent/adapter"
)

// 确保 Bridge 实现了 BridgeInterface 接口
var _ BridgeInterface = (*Bridge)(nil)

var (
	modole32          = syscall.NewLazyDLL("ole32.dll")
	modoleacc         = syscall.NewLazyDLL("oleacc.dll")
	moduser32         = syscall.NewLazyDLL("user32.dll")

	procCoInitialize        = modole32.NewProc("CoInitialize")
	procCoUninitialize      = modole32.NewProc("CoUninitialize")
	procCoCreateInstance    = modole32.NewProc("CoCreateInstance")
	procCoTaskMemFree       = modole32.NewProc("CoTaskMemFree")
	procAccessibleObjectFromWindow = modoleacc.NewProc("AccessibleObjectFromWindow")
	procFindWindow          = moduser32.NewProc("FindWindowW")
	procFindWindowEx        = moduser32.NewProc("FindWindowExW")
	procEnumWindows         = moduser32.NewProc("EnumWindows")
	procEnumChildWindows    = moduser32.NewProc("EnumChildWindows")
	procGetClassName        = moduser32.NewProc("GetClassNameW")
	procGetWindowText       = moduser32.NewProc("GetWindowTextW")
	procGetWindowTextLength = moduser32.NewProc("GetWindowTextLengthW")
)

// IAccessible COM 接口
type IAccessible struct {
lpVtbl *uintptr
}

// IAccessibleVtbl 虚函数表
type IAccessibleVtbl struct {
QueryInterface uintptr
AddRef         uintptr
Release        uintptr
GetTypeInfoCount uintptr
GetTypeInfo    uintptr
GetIDsOfNames  uintptr
Invoke         uintptr
get_accParent  uintptr
get_accChildCount uintptr
get_accChild   uintptr
get_accName    uintptr
get_accValue   uintptr
get_accDescription uintptr
get_accRole    uintptr
get_accState   uintptr
get_accHelp    uintptr
get_accHelpTopic uintptr
get_accKeyboardShortcut uintptr
get_accFocus   uintptr
get_accSelection uintptr
get_accDefaultAction uintptr
accSelect      uintptr
accLocation    uintptr
accNavigate    uintptr
accHitTest     uintptr
accDoDefaultAction uintptr
put_accName    uintptr
put_accValue   uintptr
}

// WindowInfo 窗口信息
type WindowInfo struct {
	Handle uintptr
	Class  string
	Title  string
}

// Bridge Windows UIA 桥接器
type Bridge struct {
	initialized bool
}

// NewBridge 创建桥接器实例
func NewBridge() *Bridge {
	return &Bridge{}
}

// Initialize 初始化 COM
func (b *Bridge) Initialize() adapter.Result {
	if b.initialized {
		return adapter.Result{
			Status:     adapter.StatusSuccess,
			ReasonCode: adapter.ReasonOK,
		}
	}

	// 初始化 COM
	// S_OK (0): COM initialized successfully
	// S_FALSE (1): COM was already initialized
	// RPC_E_CHANGED_MODE (0x40000004): COM was already initialized with different threading model
	ret, _, _ := procCoInitialize.Call(0)
	if ret != 0 && ret != 1 && ret != 0x40000004 {
		return adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("COM_INIT_FAILED"),
			Error:      "Failed to initialize COM",
		}
	}

	b.initialized = true
	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

// FindWindow 查找窗口
func (b *Bridge) FindWindow(className, windowName string) (uintptr, adapter.Result) {
	if !b.initialized {
		return 0, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("NOT_INITIALIZED"),
		}
	}

	var classPtr, titlePtr *uint16
	if className != "" {
		classPtr, _ = syscall.UTF16PtrFromString(className)
	}
	if windowName != "" {
		titlePtr, _ = syscall.UTF16PtrFromString(windowName)
	}

	ret, _, _ := procFindWindow.Call(
		uintptr(unsafe.Pointer(classPtr)),
		uintptr(unsafe.Pointer(titlePtr)),
	)

	if ret == 0 {
		return 0, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("WINDOW_NOT_FOUND"),
		}
	}

	return ret, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

// GetWindowInfo 获取窗口信息
func (b *Bridge) GetWindowInfo(handle uintptr) (WindowInfo, adapter.Result) {
	if handle == 0 {
		return WindowInfo{}, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("INVALID_HANDLE"),
		}
	}

	// 获取窗口标题长度
	len, _, _ := procGetWindowTextLength.Call(handle)
	if len == 0 {
		return WindowInfo{Handle: handle}, adapter.Result{
			Status:     adapter.StatusSuccess,
			ReasonCode: adapter.ReasonOK,
		}
	}

	// 获取窗口标题
	titleBuf := make([]uint16, len+1)
	procGetWindowText.Call(handle, uintptr(unsafe.Pointer(&titleBuf[0])), uintptr(len+1))
	title := syscall.UTF16ToString(titleBuf)

	// 获取窗口类名
	classBuf := make([]uint16, 256)
	procGetClassName.Call(handle, uintptr(unsafe.Pointer(&classBuf[0])), 256)
	className := syscall.UTF16ToString(classBuf)

	return WindowInfo{
		Handle: handle,
		Class:  className,
		Title:  title,
	}, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

// FindChildWindow 查找子窗口
func (b *Bridge) FindChildWindow(parentHandle uintptr, className, windowName string) (uintptr, adapter.Result) {
	if !b.initialized {
		return 0, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("NOT_INITIALIZED"),
		}
	}

	var classPtr, titlePtr *uint16
	if className != "" {
		classPtr, _ = syscall.UTF16PtrFromString(className)
	}
	if windowName != "" {
		titlePtr, _ = syscall.UTF16PtrFromString(windowName)
	}

	ret, _, _ := procFindWindowEx.Call(
		parentHandle,
		0,
		uintptr(unsafe.Pointer(classPtr)),
		uintptr(unsafe.Pointer(titlePtr)),
	)

	if ret == 0 {
		return 0, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("WINDOW_NOT_FOUND"),
		}
	}

	return ret, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

// GetAccessible 获取可访问对象
func (b *Bridge) GetAccessible(windowHandle uintptr) (*IAccessible, adapter.Result) {
	if !b.initialized {
		return nil, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("NOT_INITIALIZED"),
		}
	}

	// 定义要尝试的 OBJID 列表
	objidCandidates := []struct {
		value uintptr
		name  string
	}{
		{0x00000000, "OBJID_WINDOW"},
		{0xFFFFFFFF, "OBJID_SYSMENU"},
		{0xFFFFFFFE, "OBJID_TITLE"},
		{0xFFFFFFFD, "OBJID_MENU"},
		{0xFFFFFFFC, "OBJID_CLIENT"},
		{0xFFFFFFFB, "OBJID_VSCROLL"},
		{0xFFFFFFFA, "OBJID_HSCROLL"},
	}

	type attemptResult struct {
		objidValue uintptr
		objidName  string
		ret        uintptr
		pAcc       *IAccessible
		childCount uintptr
		success    bool
	}

	var attempts []attemptResult
	var bestAcc *IAccessible
	var bestObjidValue uintptr
	var bestObjidName string
	var bestChildCount uintptr = 0
	var bestRet uintptr
	var allEmpty bool = true
	var anySuccess bool = false

	// 尝试所有 OBJID
	for _, objid := range objidCandidates {
		var pAcc *IAccessible
		ret, _, _ := procAccessibleObjectFromWindow.Call(
			windowHandle,
			objid.value,
			uintptr(unsafe.Pointer(&IID_IAccessible)),
			uintptr(unsafe.Pointer(&pAcc)),
		)

		success := (ret == 0 && pAcc != nil)
		var childCount uintptr = 0
		if success {
			// 获取 child count
			if pAcc != nil && pAcc.lpVtbl != nil {
				vtbl := (*IAccessibleVtbl)(unsafe.Pointer(pAcc.lpVtbl))
				if vtbl.get_accChildCount != 0 {
					count, _, _ := syscall.Syscall(
						vtbl.get_accChildCount,
						1,
						uintptr(unsafe.Pointer(pAcc)),
						0,
						0,
					)
					childCount = count
				}
			}
			anySuccess = true
			if childCount > 0 {
				allEmpty = false
			}
		}

		attempts = append(attempts, attemptResult{
			objidValue: objid.value,
			objidName:  objid.name,
			ret:        ret,
			pAcc:       pAcc,
			childCount: childCount,
			success:    success,
		})

		// 选择最佳候选：优先 childCount > 0，然后选择 childCount 最大的
		if success {
			if childCount > 0 {
				if childCount > bestChildCount {
					bestAcc = pAcc
					bestObjidValue = objid.value
					bestObjidName = objid.name
					bestChildCount = childCount
					bestRet = ret
				}
			} else if bestChildCount == 0 && bestAcc == nil {
				// 如果还没有任何 childCount > 0 的候选，选择第一个成功的空树
				bestAcc = pAcc
				bestObjidValue = objid.value
				bestObjidName = objid.name
				bestChildCount = childCount
				bestRet = ret
			}
		}
	}

	// 构建诊断信息
	objidTriedList := ""
	objidResultsSummary := ""
	for i, attempt := range attempts {
		if i > 0 {
			objidTriedList += ","
			objidResultsSummary += "; "
		}
		objidTriedList += fmt.Sprintf("0x%X", attempt.objidValue)
		objidResultsSummary += fmt.Sprintf("%s:ret=0x%X,child=%d,ok=%v",
			attempt.objidName, attempt.ret, attempt.childCount, attempt.success)
	}

	// 如果没有成功，返回错误
	if !anySuccess {
		return nil, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("ACCESSIBLE_NOT_FOUND"),
			Error:      "AccessibleObjectFromWindow failed for all OBJIDs",
			Diagnostics: []adapter.Diagnostic{
				{
					Timestamp: time.Now(),
					Level:     "error",
					Message:   "AccessibleObjectFromWindow failed for all OBJIDs",
					Context: map[string]string{
						"window_handle":        strconv.FormatUint(uint64(windowHandle), 10),
						"objid_tried_list":    objidTriedList,
						"objid_results_summary": objidResultsSummary,
						"objid_attempt_count": strconv.Itoa(len(attempts)),
						"selected_objid":      "none",
						"selected_objid_name": "none",
						"selected_objid_child_count": "0",
						"all_objid_empty":     "true",
					},
				},
			},
		}
	}

	// 有成功，返回最佳候选
	selectedObjid := fmt.Sprintf("0x%X", bestObjidValue)
	selectedObjidName := bestObjidName
	selectedChildCount := strconv.FormatUint(uint64(bestChildCount), 10)
	allObjidEmpty := "false"
	if allEmpty {
		allObjidEmpty = "true"
	}

	return bestAcc, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
		Diagnostics: []adapter.Diagnostic{
			{
				Timestamp: time.Now(),
				Level:     "info",
				Message:   "AccessibleObjectFromWindow succeeded with best OBJID",
				Context: map[string]string{
					"window_handle":        strconv.FormatUint(uint64(windowHandle), 10),
					"return_code":          fmt.Sprintf("0x%X", bestRet),
					"return_code_hex":      fmt.Sprintf("0x%X", bestRet),
					"accessible_obtained":  "true",
					"pAcc_is_nil":          strconv.FormatBool(bestAcc == nil),
					"child_count":          selectedChildCount,
					"objid_tried_list":    objidTriedList,
					"objid_results_summary": objidResultsSummary,
					"objid_attempt_count": strconv.Itoa(len(attempts)),
					"selected_objid":      selectedObjid,
					"selected_objid_name": selectedObjidName,
					"selected_objid_child_count": selectedChildCount,
					"all_objid_empty":     allObjidEmpty,
					"iid_accessible":      fmt.Sprintf("%v", IID_IAccessible),
				},
			},
		},
	}
}

// FindTopLevelWindows 查找顶级窗口列表
func (b *Bridge) FindTopLevelWindows(className, windowName string) ([]uintptr, adapter.Result) {
	if !b.initialized {
		return nil, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("NOT_INITIALIZED"),
		}
	}

	// 初始化为空切片而不是 nil，确保总是返回切片
	handles := []uintptr{}

	// 使用 EnumWindows 枚举所有顶级窗口
	err := enumerateWindows(func(hwnd uintptr, lParam uintptr) uintptr {
		// 获取窗口类名
		classBuf := make([]uint16, 256)
		procGetClassName.Call(hwnd, uintptr(unsafe.Pointer(&classBuf[0])), 256)
		currentClass := syscall.UTF16ToString(classBuf)

		// 获取窗口标题
		len, _, _ := procGetWindowTextLength.Call(hwnd)
		var currentTitle string
		if len > 0 {
			titleBuf := make([]uint16, len+1)
			procGetWindowText.Call(hwnd, uintptr(unsafe.Pointer(&titleBuf[0])), uintptr(len+1))
			currentTitle = syscall.UTF16ToString(titleBuf)
		}

		// 检查是否匹配条件
		matchesClass := className == "" || currentClass == className
		matchesTitle := windowName == "" || currentTitle == windowName

		if matchesClass && matchesTitle {
			handles = append(handles, hwnd)
		}

		return 1 // 继续枚举
	})

	if err != nil {
		// EnumWindows 失败，回退到 FindWindow
		handle, result := b.FindWindow(className, windowName)
		if result.Status == adapter.StatusSuccess {
			return []uintptr{handle}, adapter.Result{
				Status:     adapter.StatusSuccess,
				ReasonCode: adapter.ReasonOK,
			}
		}
		// 如果窗口未找到，返回空列表而不是错误
		if result.ReasonCode == adapter.ReasonCode("WINDOW_NOT_FOUND") {
			return []uintptr{}, adapter.Result{
				Status:     adapter.StatusSuccess,
				ReasonCode: adapter.ReasonOK,
			}
		}
		return nil, result
	}

	return handles, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

// GetWindowText 获取窗口标题
func (b *Bridge) GetWindowText(handle uintptr) (string, adapter.Result) {
	if handle == 0 {
		return "", adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("INVALID_HANDLE"),
		}
	}

	len, _, _ := procGetWindowTextLength.Call(handle)
	if len == 0 {
		return "", adapter.Result{
			Status:     adapter.StatusSuccess,
			ReasonCode: adapter.ReasonOK,
		}
	}

	titleBuf := make([]uint16, len+1)
	procGetWindowText.Call(handle, uintptr(unsafe.Pointer(&titleBuf[0])), uintptr(len+1))
	title := syscall.UTF16ToString(titleBuf)

	return title, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

// GetWindowClass 获取窗口类名
func (b *Bridge) GetWindowClass(handle uintptr) (string, adapter.Result) {
	if handle == 0 {
		return "", adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("INVALID_HANDLE"),
		}
	}

	classBuf := make([]uint16, 256)
	procGetClassName.Call(handle, uintptr(unsafe.Pointer(&classBuf[0])), 256)
	className := syscall.UTF16ToString(classBuf)

	return className, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

// FocusWindow 聚焦到窗口
func (b *Bridge) FocusWindow(handle uintptr) adapter.Result {
	if handle == 0 {
		return adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("INVALID_HANDLE"),
		}
	}

	// 使用 user32.dll 的 SetForegroundWindow
	moduser32 := syscall.NewLazyDLL("user32.dll")
	procSetForegroundWindow := moduser32.NewProc("SetForegroundWindow")
	ret, _, _ := procSetForegroundWindow.Call(handle)

	if ret == 0 {
		return adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("FOCUS_FAILED"),
		}
	}

	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

// tryChildWindowsRecursive 递归尝试子窗口并获取可访问对象
func (b *Bridge) tryChildWindowsRecursive(parentHandle uintptr, currentDepth int, maxDepth int, visitedHandles *[]uintptr) (uintptr, *IAccessible, adapter.Result, int) {
	// 存储找到的有效窗口句柄
	var foundHandle uintptr
	var foundAcc *IAccessible
	var foundResult adapter.Result
	var foundInfo WindowInfo
	var foundDepth int = currentDepth
	var foundChildCount uintptr = 0

	// 枚举所有子窗口
	childHandles := []uintptr{}
	err := enumerateChildWindows(parentHandle, func(hwnd uintptr, lParam uintptr) uintptr {
		childHandles = append(childHandles, hwnd)
		return 1 // 继续枚举
	})

	if err != nil {
		return 0, nil, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("ENUM_CHILD_FAILED"),
			Error:      "Failed to enumerate child windows",
		}, currentDepth
	}

	// 诊断信息：记录当前深度和子窗口数量
	diagnostics := []adapter.Diagnostic{
		{
			Timestamp: time.Now(),
			Level:     "info",
			Message:   "Trying child windows at depth",
			Context: map[string]string{
				"parent_handle": strconv.FormatUint(uint64(parentHandle), 10),
				"current_depth": strconv.Itoa(currentDepth),
				"max_depth":     strconv.Itoa(maxDepth),
				"child_count":   strconv.Itoa(len(childHandles)),
			},
		},
	}

	// 记录已访问窗口
	for _, h := range childHandles {
		*visitedHandles = append(*visitedHandles, h)
	}

	// 尝试每个子窗口
	for i, childHandle := range childHandles {
		// 获取子窗口信息
		childInfo, infoResult := b.GetWindowInfo(childHandle)
		if infoResult.Status != adapter.StatusSuccess {
			continue
		}

		// 尝试获取可访问对象
		pAcc, accResult := b.GetAccessible(childHandle)
		childCount := uintptr(0)
		if accResult.Status == adapter.StatusSuccess {
			// 检查是否有有效的子节点
			childCount = b.getAccChildCount(pAcc)

			// 选择最佳窗口：优先选择有子节点的，同等情况选择深度最浅的
			isBetter := false
			if foundAcc == nil {
				isBetter = true
			} else if childCount > 0 && foundChildCount == 0 {
				isBetter = true
			} else if childCount > 0 && foundChildCount > 0 && currentDepth < foundDepth {
				isBetter = true
			} else if childCount == 0 && foundChildCount == 0 && currentDepth < foundDepth {
				isBetter = true
			}

			if isBetter {
				foundHandle = childHandle
				foundAcc = pAcc
				foundResult = accResult
				foundInfo = childInfo
				foundDepth = currentDepth
				foundChildCount = childCount

				// 添加成功诊断
				successReason := "accessible_with_children"
				if childCount == 0 {
					successReason = "accessible_no_children"
				}

				diagnostics = append(diagnostics, adapter.Diagnostic{
					Timestamp: time.Now(),
					Level:     "info",
					Message:   "Found accessible window",
					Context: map[string]string{
						"window_handle": strconv.FormatUint(uint64(childHandle), 10),
						"window_class":  childInfo.Class,
						"window_title":  childInfo.Title,
						"window_index":  strconv.Itoa(i),
						"child_count":   strconv.FormatUint(uint64(childCount), 10),
						"depth":         strconv.Itoa(currentDepth),
						"success_reason": successReason,
						"has_children":   strconv.FormatBool(childCount > 0),
					},
				})

				// 将accResult的诊断信息也添加进来
				diagnostics = append(diagnostics, accResult.Diagnostics...)
			}

			// 记录 childCount == 0 的情况用于诊断
			if childCount == 0 {
				diagnostics = append(diagnostics, adapter.Diagnostic{
					Timestamp: time.Now(),
					Level:     "debug",
					Message:   "Window has no children",
					Context: map[string]string{
						"window_handle": strconv.FormatUint(uint64(childHandle), 10),
						"window_class":  childInfo.Class,
						"window_title":  childInfo.Title,
						"window_index":  strconv.Itoa(i),
						"depth":         strconv.Itoa(currentDepth),
						"child_count":   "0",
						"accessible":    "true",
						"note":          "Window is accessible but has no child nodes",
					},
				})
			}
		}

		// 记录尝试日志
		diagnostics = append(diagnostics, adapter.Diagnostic{
			Timestamp: time.Now(),
			Level:     "debug",
			Message:   "Tried window",
			Context: map[string]string{
				"window_handle": strconv.FormatUint(uint64(childHandle), 10),
				"window_class":  childInfo.Class,
				"window_title":  childInfo.Title,
				"window_index":  strconv.Itoa(i),
				"depth":         strconv.Itoa(currentDepth),
				"accessible":    strconv.FormatBool(accResult.Status == adapter.StatusSuccess),
				"error":         accResult.Error,
			},
		})

		// 如果当前窗口没有子节点，且未达到最大深度，继续递归搜索
		if accResult.Status == adapter.StatusSuccess && childCount == 0 && currentDepth < maxDepth {
			grandchildHandle, grandchildAcc, grandchildResult, grandchildDepth := b.tryChildWindowsRecursive(childHandle, currentDepth+1, maxDepth, visitedHandles)
			if grandchildResult.Status == adapter.StatusSuccess && grandchildAcc != nil {
				grandchildCount := b.getAccChildCount(grandchildAcc)

				// 检查孙窗口是否更好
				isGrandchildBetter := false
				if foundAcc == nil {
					isGrandchildBetter = true
				} else if grandchildCount > 0 && foundChildCount == 0 {
					isGrandchildBetter = true
				} else if grandchildCount > 0 && foundChildCount > 0 && grandchildDepth < foundDepth {
					isGrandchildBetter = true
				} else if grandchildCount == 0 && foundChildCount == 0 && grandchildDepth < foundDepth {
					isGrandchildBetter = true
				}

				if isGrandchildBetter {
					foundHandle = grandchildHandle
					foundAcc = grandchildAcc
					foundResult = grandchildResult
					foundDepth = grandchildDepth
					foundChildCount = grandchildCount
					// 注意：这里不更新foundInfo，因为孙窗口的信息可能不同
				}
			}
		}
	}

	// 检查是否找到了可访问对象
	if foundAcc != nil {
		// 获取窗口信息（如果之前没有获取）
		if foundInfo.Class == "" && foundInfo.Title == "" {
			foundInfo, _ = b.GetWindowInfo(foundHandle)
		}

		// 添加最终诊断
		finalChildCount := b.getAccChildCount(foundAcc)
		diagnostics = append(diagnostics, adapter.Diagnostic{
			Timestamp: time.Now(),
			Level:     "info",
			Message:   "Selected best window for accessible tree",
			Context: map[string]string{
				"selected_handle":    strconv.FormatUint(uint64(foundHandle), 10),
				"selected_class":     foundInfo.Class,
				"selected_title":     foundInfo.Title,
				"selected_depth":     strconv.Itoa(foundDepth),
				"child_count":        strconv.FormatUint(uint64(finalChildCount), 10),
				"has_children":       strconv.FormatBool(finalChildCount > 0),
				"selection_method":   "recursive_depth_first",
				"search_depth":       strconv.Itoa(maxDepth),
				"visited_at_depth":   strconv.Itoa(currentDepth),
			},
		})

		foundResult.Diagnostics = diagnostics
		return foundHandle, foundAcc, foundResult, foundDepth
	}

	// 如果没有找到有效的窗口
	return 0, nil, adapter.Result{
		Status:     adapter.StatusFailed,
		ReasonCode: adapter.ReasonCode("NO_VALID_CHILD"),
		Error:      fmt.Sprintf("No accessible window found up to depth %d", maxDepth),
		Diagnostics: diagnostics,
	}, currentDepth
}

// tryChildWindows 尝试枚举子窗口并获取可访问对象
func (b *Bridge) tryChildWindows(parentHandle uintptr) (uintptr, *IAccessible, adapter.Result) {
	// 使用递归搜索，最大深度为3
	maxDepth := 3
	visitedHandles := []uintptr{}

	// 调用递归搜索函数
	foundHandle, foundAcc, foundResult, selectedDepth := b.tryChildWindowsRecursive(parentHandle, 0, maxDepth, &visitedHandles)

	if foundResult.Status == adapter.StatusSuccess && foundAcc != nil {
		// 获取窗口信息
		foundInfo, _ := b.GetWindowInfo(foundHandle)
		finalChildCount := b.getAccChildCount(foundAcc)

		// 添加最终诊断信息，包含要求的字段
		selectedHandlePath := strconv.FormatUint(uint64(parentHandle), 10)
		if selectedDepth > 0 {
			selectedHandlePath = fmt.Sprintf("%s->%d", selectedHandlePath, foundHandle)
		}

		// 更新foundResult的诊断信息
		finalDiagnostics := foundResult.Diagnostics
		finalDiagnostics = append(finalDiagnostics, adapter.Diagnostic{
			Timestamp: time.Now(),
			Level:     "info",
			Message:   "tryChildWindows completed with recursive search",
			Context: map[string]string{
				"parent_handle":       strconv.FormatUint(uint64(parentHandle), 10),
				"selected_handle":     strconv.FormatUint(uint64(foundHandle), 10),
				"selected_class":      foundInfo.Class,
				"selected_title":      foundInfo.Title,
				"search_depth":        strconv.Itoa(maxDepth),
				"selected_depth":      strconv.Itoa(selectedDepth),
				"selected_handle_path": selectedHandlePath,
				"visited_window_count": strconv.Itoa(len(visitedHandles)),
				"selected_child_count": strconv.FormatUint(uint64(finalChildCount), 10),
				"has_children":        strconv.FormatBool(finalChildCount > 0),
				"selection_method":    "recursive_depth_first",
			},
		})

		foundResult.Diagnostics = finalDiagnostics
		return foundHandle, foundAcc, foundResult
	}

	// 如果没有找到有效的窗口，返回原始结果（递归函数已经包含了诊断信息）
	return foundHandle, foundAcc, foundResult
}

// EnumerateAccessibleNodes 枚举可访问节点
func (b *Bridge) EnumerateAccessibleNodes(windowHandle uintptr) ([]AccessibleNode, adapter.Result) {
	if !b.initialized {
		return nil, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("NOT_INITIALIZED"),
		}
	}

	// 获取窗口信息用于诊断
	info, infoResult := b.GetWindowInfo(windowHandle)
	if infoResult.Status != adapter.StatusSuccess {
		// 对于无效句柄，返回空列表而不是错误
		// 这样可以让适配器继续工作
		return []AccessibleNode{}, adapter.Result{
			Status:     adapter.StatusSuccess,
			ReasonCode: adapter.ReasonOK,
		}
	}

	// 尝试获取可访问对象
	pAcc, result := b.GetAccessible(windowHandle)
	effectiveHandle := windowHandle
	triedChildWindows := false
	childWindowSelected := false
	switchReason := ""
	topLevelChildCount := uintptr(0)

	if result.Status != adapter.StatusSuccess {
		// 失败时尝试子窗口
		childHandle, childAcc, childResult := b.tryChildWindows(windowHandle)
		triedChildWindows = true
		if childResult.Status == adapter.StatusSuccess && childAcc != nil {
			effectiveHandle = childHandle
			pAcc = childAcc
			result = childResult
			childWindowSelected = true
			switchReason = "top_level_accessible_failed"

			// 更新窗口信息为子窗口信息
			childInfo, childInfoResult := b.GetWindowInfo(effectiveHandle)
			if childInfoResult.Status == adapter.StatusSuccess {
				info = childInfo
			}

			// 添加诊断信息表明我们使用了子窗口
			childResult.Diagnostics = append(childResult.Diagnostics, adapter.Diagnostic{
				Timestamp: time.Now(),
				Level:     "info",
				Message:   "Using child window instead of parent",
				Context: map[string]string{
					"parent_handle": strconv.FormatUint(uint64(windowHandle), 10),
					"child_handle":  strconv.FormatUint(uint64(effectiveHandle), 10),
					"child_class":   info.Class,
					"child_title":   info.Title,
					"switch_reason": switchReason,
				},
			})
		} else {
			// 子窗口也失败，返回基本窗口节点
			nodes := []AccessibleNode{
				{
					Handle:    windowHandle,
					Name:      info.Title,
					Role:      "window",
					ClassName: info.Class,
				},
			}

			// 收集所有诊断信息
			diagnostics := []adapter.Diagnostic{}
			// 添加GetAccessible的诊断信息
			diagnostics = append(diagnostics, result.Diagnostics...)
			// 添加子窗口尝试的诊断信息
			if childResult.Diagnostics != nil {
				diagnostics = append(diagnostics, childResult.Diagnostics...)
			}
			// 添加fallback诊断
			fallbackReason := "accessible_not_found_no_child"
			if result.ReasonCode == adapter.ReasonCode("ACCESSIBLE_NOT_FOUND") {
				if result.Diagnostics != nil && len(result.Diagnostics) > 0 {
					for _, diag := range result.Diagnostics {
						if diag.Context != nil && diag.Context["return_code_hex"] != "" {
							fallbackReason = fmt.Sprintf("accessible_object_from_window_failed_%s", diag.Context["return_code_hex"])
							break
						}
					}
				}
			}

			// 尝试获取子窗口数量信息
			childWindowCount := "0"
			if childResult.Diagnostics != nil {
				for _, diag := range childResult.Diagnostics {
					if diag.Context != nil && diag.Context["child_count"] != "" {
						childWindowCount = diag.Context["child_count"]
						break
					}
				}
			}

			diagnostics = append(diagnostics, adapter.Diagnostic{
				Timestamp: time.Now(),
				Level:     "warning",
				Message:   "GetAccessible failed, falling back to window node",
				Context: map[string]string{
					"window_handle":              strconv.FormatUint(uint64(windowHandle), 10),
					"window_class":               info.Class,
					"window_title":               info.Title,
					"error":                      result.Error,
					"reason_code":                string(result.ReasonCode),
					"fallback_reason":            fallbackReason,
					"accessible_object_obtained": "false",
					"child_count":                "0",
					"child_window_tried":         "true",
					"child_window_found":         "false",
					"total_child_windows":        childWindowCount,
					"objid_client":               "0xFFFFFFFC",
					"bridge_issue":               "true",
					"bridge_layer_blocked":       "true",
					"diagnostic_summary":         fmt.Sprintf("AccessibleObjectFromWindow failed with code %s, no valid child window found", result.ReasonCode),
					"tried_child_windows":        "true",
					"child_window_selected":      "false",
					"effective_window_handle":    strconv.FormatUint(uint64(windowHandle), 10),
					"switch_reason":              "top_level_accessible_failed_no_child",
				},
			})

			return nodes, adapter.Result{
				Status:     adapter.StatusSuccess,
				ReasonCode: adapter.ReasonOK,
				Diagnostics: diagnostics,
			}
		}
	} else {
		// GetAccessible 成功，检查子节点数量
		topLevelChildCount = b.getAccChildCount(pAcc)
		if topLevelChildCount == 0 {
			// 成功但空树，尝试子窗口
			childHandle, childAcc, childResult := b.tryChildWindows(windowHandle)
			triedChildWindows = true
			if childResult.Status == adapter.StatusSuccess && childAcc != nil {
				effectiveHandle = childHandle
				pAcc = childAcc
				result = childResult
				childWindowSelected = true
				switchReason = "top_level_accessible_but_empty"

				// 更新窗口信息为子窗口信息
				childInfo, childInfoResult := b.GetWindowInfo(effectiveHandle)
				if childInfoResult.Status == adapter.StatusSuccess {
					info = childInfo
				}

				// 添加诊断信息表明我们使用了子窗口
				childResult.Diagnostics = append(childResult.Diagnostics, adapter.Diagnostic{
					Timestamp: time.Now(),
					Level:     "info",
					Message:   "Using child window because top-level accessible tree is empty",
					Context: map[string]string{
						"parent_handle": strconv.FormatUint(uint64(windowHandle), 10),
						"child_handle":  strconv.FormatUint(uint64(effectiveHandle), 10),
						"child_class":   info.Class,
						"child_title":   info.Title,
						"switch_reason": switchReason,
						"top_level_child_count": "0",
					},
				})
			}
		}
	}

	// 获取根节点的子节点数量
	childCount := b.getAccChildCount(pAcc)

	// 创建根节点
	rootNode := AccessibleNode{
		Handle:    effectiveHandle,
		Name:      info.Title,
		Role:      "window",
		ClassName: info.Class,
	}

	// 递归遍历子节点
	children := b.enumerateAccessibleChildren(pAcc, 0, 1)
	rootNode.Children = children

	// 收集诊断信息
	diagnostics := []adapter.Diagnostic{}
	// 添加GetAccessible的诊断信息
	diagnostics = append(diagnostics, result.Diagnostics...)

	// 扁平化所有节点用于统计
	allNodes := []AccessibleNode{rootNode}
	for _, child := range children {
		allNodes = append(allNodes, child)
		// 递归收集所有子节点（简化版）
		allNodes = append(allNodes, flattenNodeChildren(child)...)
	}

	diagnostics = append(diagnostics, adapter.Diagnostic{
		Timestamp: time.Now(),
		Level:     "info",
		Message:   "EnumerateAccessibleNodes succeeded",
		Context: map[string]string{
			"window_handle":           strconv.FormatUint(uint64(windowHandle), 10),
			"effective_window_handle": strconv.FormatUint(uint64(effectiveHandle), 10),
			"window_class":            info.Class,
			"window_title":            info.Title,
			"accessible_obtained":     "true",
			"root_child_count":        strconv.FormatUint(uint64(childCount), 10),
			"children_enumerated":     strconv.Itoa(len(children)),
			"total_nodes_count":       strconv.Itoa(len(allNodes)),
			"fallback_used":           strconv.FormatBool(childWindowSelected),
			"root_name":               info.Title,
			"root_role":               "window",
			"objid_client":            "0xFFFFFFFC",
			"bridge_layer_status":     "success",
			"diagnostic_summary":      fmt.Sprintf("Got accessible subtree with %d total nodes, root child count: %d", len(allNodes), childCount),
			"top_level_child_count":   strconv.FormatUint(uint64(topLevelChildCount), 10),
			"tried_child_windows":     strconv.FormatBool(triedChildWindows),
			"child_window_selected":   strconv.FormatBool(childWindowSelected),
			"effective_child_count":   strconv.FormatUint(uint64(childCount), 10),
			"switch_reason":           switchReason,
		},
	})

	return []AccessibleNode{rootNode}, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
		Diagnostics: diagnostics,
	}
}

// enumerateAccessibleChildren 递归枚举可访问子节点
func (b *Bridge) enumerateAccessibleChildren(pAcc *IAccessible, childID uintptr, depth int) []AccessibleNode {
	if depth > 10 { // 限制递归深度，防止无限循环
		return nil
	}

	// 获取子节点数量
	childCount := b.getAccChildCount(pAcc)
	if childCount == 0 {
		return nil
	}

	nodes := []AccessibleNode{}

	// 遍历所有子节点
	for i := uintptr(1); i <= childCount; i++ {
		// 获取子对象
		childAcc, childIDResult := b.getAccChild(pAcc, i)
		if childIDResult.Status != adapter.StatusSuccess {
			continue
		}

		// 获取子节点信息
		node := b.getAccessibleNodeInfo(childAcc, i)
		if node.Name != "" || node.Role != "" {
			// 递归获取子节点的子节点
			node.Children = b.enumerateAccessibleChildren(childAcc, i, depth+1)
			nodes = append(nodes, node)
		}
	}

	return nodes
}

// getAccChildCount 获取子节点数量
func (b *Bridge) getAccChildCount(pAcc *IAccessible) uintptr {
	if pAcc == nil || pAcc.lpVtbl == nil {
		return 0
	}

	vtbl := (*IAccessibleVtbl)(unsafe.Pointer(pAcc.lpVtbl))
	if vtbl.get_accChildCount == 0 {
		return 0
	}

	// 调用 get_accChildCount
	ret, _, _ := syscall.Syscall(
		vtbl.get_accChildCount,
		1,
		uintptr(unsafe.Pointer(pAcc)),
		0,
		0,
	)

	// 返回值在 eax 中，ret 包含 child count
	return ret
}

// getAccChild 获取指定 ID 的子对象
func (b *Bridge) getAccChild(pAcc *IAccessible, childID uintptr) (*IAccessible, adapter.Result) {
	if pAcc == nil || pAcc.lpVtbl == nil {
		return nil, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("INVALID_ACCESSIBLE"),
		}
	}

	vtbl := (*IAccessibleVtbl)(unsafe.Pointer(pAcc.lpVtbl))
	if vtbl.get_accChild == 0 {
		return nil, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("METHOD_NOT_SUPPORTED"),
		}
	}

	// VARIANT 结构用于传递 child ID
	type VARIANT struct {
		Vt        uint16
		Reserved1 uint16
		Reserved2 uint16
		Reserved3 uint16
		Data      [8]byte
	}

	var variant VARIANT
	variant.Vt = 3 // VT_I4 (integer)
	*(*uintptr)(unsafe.Pointer(&variant.Data[0])) = childID

	var pChild *IAccessible

	// 调用 get_accChild
	ret, _, _ := syscall.Syscall(
		vtbl.get_accChild,
		3,
		uintptr(unsafe.Pointer(pAcc)),
		uintptr(unsafe.Pointer(&variant)),
		uintptr(unsafe.Pointer(&pChild)),
	)

	if ret != 0 || pChild == nil {
		return nil, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("CHILD_NOT_FOUND"),
		}
	}

	return pChild, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

// getAccessibleNodeInfo 从 IAccessible 对象获取节点信息
func (b *Bridge) getAccessibleNodeInfo(pAcc *IAccessible, childID uintptr) AccessibleNode {
	node := AccessibleNode{}

	if pAcc == nil || pAcc.lpVtbl == nil {
		return node
	}

	vtbl := (*IAccessibleVtbl)(unsafe.Pointer(pAcc.lpVtbl))

	// 获取名称 (get_accName)
	if vtbl.get_accName != 0 {
		var namePtr *uint16
		type VARIANT struct {
			Vt        uint16
			Reserved1 uint16
			Reserved2 uint16
			Reserved3 uint16
			Data      [8]byte
		}
		var variant VARIANT
		variant.Vt = 3 // VT_I4
		*(*uintptr)(unsafe.Pointer(&variant.Data[0])) = childID

		ret, _, _ := syscall.Syscall(
			vtbl.get_accName,
			3,
			uintptr(unsafe.Pointer(pAcc)),
			uintptr(unsafe.Pointer(&variant)),
			uintptr(unsafe.Pointer(&namePtr)),
		)

		if ret == 0 && namePtr != nil {
			node.Name = syscall.UTF16ToString((*[1 << 20]uint16)(unsafe.Pointer(namePtr))[:])
			// 释放 BSTR
			modole32 := syscall.NewLazyDLL("ole32.dll")
			procSysFreeString := modole32.NewProc("SysFreeString")
			procSysFreeString.Call(uintptr(unsafe.Pointer(namePtr)))
		}
	}

	// 获取角色 (get_accRole)
	if vtbl.get_accRole != 0 {
		type VARIANT struct {
			Vt        uint16
			Reserved1 uint16
			Reserved2 uint16
			Reserved3 uint16
			Data      [8]byte
		}
		var variant VARIANT
		variant.Vt = 3 // VT_I4
		*(*uintptr)(unsafe.Pointer(&variant.Data[0])) = childID

		var roleVariant VARIANT
		ret, _, _ := syscall.Syscall(
			vtbl.get_accRole,
			3,
			uintptr(unsafe.Pointer(pAcc)),
			uintptr(unsafe.Pointer(&variant)),
			uintptr(unsafe.Pointer(&roleVariant)),
		)

		if ret == 0 && roleVariant.Vt == 3 {
			roleValue := *(*uintptr)(unsafe.Pointer(&roleVariant.Data[0]))
			node.Role = b.getRoleString(roleValue)
		}
	}

	// 获取类名 (通过 get_accClassName 或其他方式)
	// IAccessible 没有直接的 get_accClassName，需要从其他属性推断
	node.ClassName = ""

	// 获取位置信息 (accLocation)
	if vtbl.accLocation != 0 {
		type VARIANT struct {
			Vt        uint16
			Reserved1 uint16
			Reserved2 uint16
			Reserved3 uint16
			Data      [8]byte
		}
		var variant VARIANT
		variant.Vt = 3 // VT_I4
		*(*uintptr)(unsafe.Pointer(&variant.Data[0])) = childID

		var left, top, width, height int32
		ret, _, _ := syscall.Syscall6(
			vtbl.accLocation,
			6,
			uintptr(unsafe.Pointer(pAcc)),
			uintptr(unsafe.Pointer(&left)),
			uintptr(unsafe.Pointer(&top)),
			uintptr(unsafe.Pointer(&width)),
			uintptr(unsafe.Pointer(&height)),
			uintptr(unsafe.Pointer(&variant)),
		)

		if ret == 0 {
			node.Bounds = [4]int{int(left), int(top), int(width), int(height)}
		}
	}

	return node
}

// getRoleString 将角色值转换为字符串
func (b *Bridge) getRoleString(roleValue uintptr) string {
	// 参考: https://docs.microsoft.com/en-us/windows/win32/winauto/object-roles
	switch roleValue {
	case 1:
		return "titlebar"
	case 2:
		return "menubar"
	case 3:
		return "scrollbar"
	case 4:
		return "grip"
	case 5:
		return "sound"
	case 6:
		return "cursor"
	case 7:
		return "caret"
	case 8:
		return "alert"
	case 9:
		return "window"
	case 10:
		return "client"
	case 11:
		return "popupmenu"
	case 12:
		return "menuitem"
	case 13:
		return "tooltip"
	case 14:
		return "application"
	case 15:
		return "document"
	case 16:
		return "pane"
	case 17:
		return "chart"
	case 18:
		return "dialog"
	case 19:
		return "border"
	case 20:
		return "grouping"
	case 21:
		return "separator"
	case 22:
		return "toolbar"
	case 23:
		return "statusbar"
	case 24:
		return "table"
	case 25:
		return "columnheader"
	case 26:
		return "rowheader"
	case 27:
		return "row"
	case 28:
		return "column"
	case 29:
		return "cell"
	case 30:
		return "link"
	case 31:
		return "helpballoon"
	case 32:
		return "character"
	case 33:
		return "list"
	case 34:
		return "listitem"
	case 35:
		return "outline"
	case 36:
		return "outlineitem"
	case 37:
		return "pagetab"
	case 38:
		return "propertypage"
	case 39:
		return "indicator"
	case 40:
		return "graphic"
	case 41:
		return "statictext"
	case 42:
		return "text"
	case 43:
		return "pushbutton"
	case 44:
		return "checkbutton"
	case 45:
		return "radiobutton"
	case 46:
		return "combobox"
	case 47:
		return "dropdownlist"
	case 48:
		return "progressbar"
	case 49:
		return "slider"
	case 50:
		return "spinbutton"
	case 51:
		return "diagram"
	case 52:
		return "animation"
	case 53:
		return "equation"
	case 54:
		return "buttondropdown"
	case 55:
		return "buttonmenu"
	case 56:
		return "buttondropdowngrid"
	case 57:
		return "whitespace"
	case 58:
		return "pagetablist"
	case 59:
		return "clock"
	case 60:
		return "splitbutton"
	case 61:
		return "ipaddress"
	case 62:
		return "outlinebutton"
	default:
		return "unknown"
	}
}

// CaptureWindow 截图窗口
func (b *Bridge) CaptureWindow(handle uintptr) ([]byte, adapter.Result) {
	if handle == 0 {
		return nil, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("INVALID_HANDLE"),
		}
	}

	// 使用 user32.dll 和 gdi32.dll 截图
	moduser32 := syscall.NewLazyDLL("user32.dll")
	modgdi32 := syscall.NewLazyDLL("gdi32.dll")

	// 获取窗口设备上下文 (DC)
	procGetDC := moduser32.NewProc("GetDC")
	procReleaseDC := moduser32.NewProc("ReleaseDC")
	procGetWindowRect := moduser32.NewProc("GetWindowRect")

	// 获取窗口矩形
	type RECT struct {
		Left   int32
		Top    int32
		Right  int32
		Bottom int32
	}
	var rect RECT
	ret, _, _ := procGetWindowRect.Call(handle, uintptr(unsafe.Pointer(&rect)))
	if ret == 0 {
		return nil, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("CAPTURE_FAILED"),
			Error:      "Failed to get window rectangle",
		}
	}

	width := rect.Right - rect.Left
	height := rect.Bottom - rect.Top

	// 获取窗口 DC
	hdc, _, _ := procGetDC.Call(handle)
	if hdc == 0 {
		return nil, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("CAPTURE_FAILED"),
			Error:      "Failed to get device context",
		}
	}
	defer procReleaseDC.Call(handle, hdc)

	// 创建内存 DC
	procCreateCompatibleDC := modgdi32.NewProc("CreateCompatibleDC")
	procCreateCompatibleBitmap := modgdi32.NewProc("CreateCompatibleBitmap")
	procSelectObject := modgdi32.NewProc("SelectObject")
	procBitBlt := modgdi32.NewProc("BitBlt")
	procDeleteDC := modgdi32.NewProc("DeleteDC")
	procDeleteObject := modgdi32.NewProc("DeleteObject")

	memDC, _, _ := procCreateCompatibleDC.Call(hdc)
	if memDC == 0 {
		return nil, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("CAPTURE_FAILED"),
			Error:      "Failed to create memory DC",
		}
	}
	defer procDeleteDC.Call(memDC)

	// 创建位图
	bitmap, _, _ := procCreateCompatibleBitmap.Call(hdc, uintptr(width), uintptr(height))
	if bitmap == 0 {
		return nil, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("CAPTURE_FAILED"),
			Error:      "Failed to create bitmap",
		}
	}
	defer procDeleteObject.Call(bitmap)

	// 选择位图到内存 DC
	oldBitmap, _, _ := procSelectObject.Call(memDC, bitmap)
	defer procSelectObject.Call(memDC, oldBitmap)

	// 从窗口 DC 复制到内存 DC
	// SRCCOPY = 0x00CC0020
	ret, _, _ = procBitBlt.Call(memDC, 0, 0, uintptr(width), uintptr(height), hdc, 0, 0, 0x00CC0020)
	if ret == 0 {
		return nil, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("CAPTURE_FAILED"),
			Error:      "Failed to copy bitmap",
		}
	}

	// 获取位图信息
	procGetDIBits := modgdi32.NewProc("GetDIBits")

	// BITMAPINFOHEADER 结构
	type BITMAPINFOHEADER struct {
		BiSize          uint32
		BiWidth         int32
		BiHeight        int32
		BiPlanes        uint16
		BiBitCount      uint16
		BiCompression   uint32
		BiSizeImage     uint32
		BiXPelsPerMeter int32
		BiYPelsPerMeter int32
		BiClrUsed       uint32
		BiClrImportant  uint32
	}

	bih := BITMAPINFOHEADER{
		BiSize:     uint32(unsafe.Sizeof(BITMAPINFOHEADER{})),
		BiWidth:    width,
		BiHeight:   -height, // 负高度表示从上到下的位图
		BiPlanes:   1,
		BiBitCount: 24, // 24 位 RGB
		BiCompression: 0, // BI_RGB
	}

	// 计算行大小（4 字节对齐）
	rowSize := ((int32(width)*24 + 31) / 32) * 4
	imageSize := uint32(rowSize * height)

	// 分配缓冲区
	pixels := make([]byte, imageSize)

	// 获取位图数据
	ret, _, _ = procGetDIBits.Call(memDC, bitmap, 0, uintptr(height), uintptr(unsafe.Pointer(&pixels[0])), uintptr(unsafe.Pointer(&bih)), 0)
	if ret == 0 {
		return nil, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("CAPTURE_FAILED"),
			Error:      "Failed to get bitmap bits",
		}
	}

	// 转换为 PNG 或返回原始像素数据
	// 这里返回原始像素数据（BGR 格式）
	return pixels, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

// SendKeys 发送按键
func (b *Bridge) SendKeys(handle uintptr, keys string) adapter.Result {
	if handle == 0 {
		return adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("INVALID_HANDLE"),
		}
	}

	if !b.initialized {
		return adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("NOT_INITIALIZED"),
		}
	}

	// 聚焦到窗口
	focusResult := b.FocusWindow(handle)
	if focusResult.Status != adapter.StatusSuccess {
		return focusResult
	}

	// 使用 user32.dll 的 keybd_event 发送按键
	moduser32 := syscall.NewLazyDLL("user32.dll")
	procKeybdEvent := moduser32.NewProc("keybd_event")

	// 处理特殊键序列
	if keys == "{ENTER}" {
		// 发送 Enter 键
		procKeybdEvent.Call(0x0D, 0, 0, 0) // VK_RETURN 按下
		procKeybdEvent.Call(0x0D, 0, 2, 0) // VK_RETURN 释放
		return adapter.Result{
			Status:     adapter.StatusSuccess,
			ReasonCode: adapter.ReasonOK,
		}
	}

	// 处理 Ctrl+V 组合键
	if keys == "^v" || keys == "^V" {
		// 先按下 Ctrl
		procKeybdEvent.Call(0x11, 0, 0, 0) // VK_CONTROL 按下
		// 再按下 V
		procKeybdEvent.Call(0x56, 0, 0, 0) // VK_V 按下
		// 释放 V
		procKeybdEvent.Call(0x56, 0, 2, 0) // VK_V 释放
		// 释放 Ctrl
		procKeybdEvent.Call(0x11, 0, 2, 0) // VK_CONTROL 释放
		return adapter.Result{
			Status:     adapter.StatusSuccess,
			ReasonCode: adapter.ReasonOK,
		}
	}

	// 对于普通字符，直接发送（简化实现）
	// 注意：这只是一个基础实现，实际使用时可能需要更复杂的按键映射
	for _, char := range keys {
		// 跳过特殊字符
		if char == '{' || char == '}' {
			continue
		}
		// 简单的 ASCII 到虚拟键码映射（仅支持字母和数字）
		if char >= 'a' && char <= 'z' {
			vkCode := uintptr(char - 'a' + 0x41) // A=0x41
			procKeybdEvent.Call(vkCode, 0, 0, 0)
			procKeybdEvent.Call(vkCode, 0, 2, 0)
		} else if char >= 'A' && char <= 'Z' {
			vkCode := uintptr(char - 'A' + 0x41)
			procKeybdEvent.Call(vkCode, 0, 0, 0)
			procKeybdEvent.Call(vkCode, 0, 2, 0)
		} else if char >= '0' && char <= '9' {
			vkCode := uintptr(char - '0' + 0x30) // 0=0x30
			procKeybdEvent.Call(vkCode, 0, 0, 0)
			procKeybdEvent.Call(vkCode, 0, 2, 0)
		}
	}

	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

// Click 点击窗口位置
func (b *Bridge) Click(handle uintptr, x, y int) adapter.Result {
	if handle == 0 {
		return adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("INVALID_HANDLE"),
		}
	}

	if !b.initialized {
		return adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("NOT_INITIALIZED"),
		}
	}

	// 聚焦到窗口
	focusResult := b.FocusWindow(handle)
	if focusResult.Status != adapter.StatusSuccess {
		return focusResult
	}

	// 使用 user32.dll 的 mouse_event 发送鼠标点击
	moduser32 := syscall.NewLazyDLL("user32.dll")
	procSetCursorPos := moduser32.NewProc("SetCursorPos")
	procmouse_event := moduser32.NewProc("mouse_event")

	// 获取窗口的屏幕坐标
	// 首先将窗口坐标转换为屏幕坐标
	procClientToScreen := moduser32.NewProc("ClientToScreen")

	// 创建 POINT 结构
	type POINT struct {
		X int32
		Y int32
	}
	point := POINT{X: int32(x), Y: int32(y)}

	// 调用 ClientToScreen 转换坐标
	ret, _, _ := procClientToScreen.Call(handle, uintptr(unsafe.Pointer(&point)))
	if ret == 0 {
		// 如果转换失败，使用相对坐标
		point.X = int32(x)
		point.Y = int32(y)
	}

	// 设置鼠标位置
	procSetCursorPos.Call(uintptr(point.X), uintptr(point.Y))

	// 发送鼠标左键按下事件
	// MOUSEEVENTF_LEFTDOWN = 0x0002
	procmouse_event.Call(0x0002, 0, 0, 0, 0)

	// 发送鼠标左键释放事件
	// MOUSEEVENTF_LEFTUP = 0x0004
	procmouse_event.Call(0x0004, 0, 0, 0, 0)

	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

// SetClipboardText 设置剪贴板文本
func (b *Bridge) SetClipboardText(text string) adapter.Result {
	moduser32 := syscall.NewLazyDLL("user32.dll")
	procOpenClipboard := moduser32.NewProc("OpenClipboard")
	procEmptyClipboard := moduser32.NewProc("EmptyClipboard")
	procSetClipboardData := moduser32.NewProc("SetClipboardData")
	procCloseClipboard := moduser32.NewProc("CloseClipboard")

	// 打开剪贴板
	ret, _, _ := procOpenClipboard.Call(0)
	if ret == 0 {
		return adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("CLIPBOARD_OPEN_FAILED"),
			Error:      "Failed to open clipboard",
		}
	}
	defer procCloseClipboard.Call()

	// 清空剪贴板
	ret, _, _ = procEmptyClipboard.Call()
	if ret == 0 {
		return adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("CLIPBOARD_EMPTY_FAILED"),
			Error:      "Failed to empty clipboard",
		}
	}

	// 分配全局内存并复制文本
	modkernel32 := syscall.NewLazyDLL("kernel32.dll")
	procGlobalAlloc := modkernel32.NewProc("GlobalAlloc")
	procGlobalLock := modkernel32.NewProc("GlobalLock")
	procGlobalUnlock := modkernel32.NewProc("GlobalUnlock")
	procRtlCopyMemory := modkernel32.NewProc("RtlCopyMemory")

	// 将文本转换为 UTF-16
	utf16Text := syscall.StringToUTF16(text)
	textSize := len(utf16Text) * 2 // 每个字符 2 字节

	// 分配全局内存 (GMEM_MOVEABLE = 0x0002)
	hMem, _, _ := procGlobalAlloc.Call(0x0002, uintptr(textSize))
	if hMem == 0 {
		return adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("MEMORY_ALLOC_FAILED"),
			Error:      "Failed to allocate global memory",
		}
	}

	// 锁定内存
	ptr, _, _ := procGlobalLock.Call(hMem)
	if ptr == 0 {
		procGlobalAlloc.Call(hMem) // 释放内存
		return adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("MEMORY_LOCK_FAILED"),
			Error:      "Failed to lock global memory",
		}
	}

	// 复制文本到内存
	procRtlCopyMemory.Call(ptr, uintptr(unsafe.Pointer(&utf16Text[0])), uintptr(textSize))

	// 解锁内存
	procGlobalUnlock.Call(hMem)

	// 设置剪贴板数据 (CF_UNICODETEXT = 13)
	ret, _, _ = procSetClipboardData.Call(13, hMem)
	if ret == 0 {
		procGlobalAlloc.Call(hMem) // 释放内存
		return adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("CLIPBOARD_SET_FAILED"),
			Error:      "Failed to set clipboard data",
		}
	}

	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

// GetClipboardText 获取剪贴板文本
func (b *Bridge) GetClipboardText() (string, adapter.Result) {
	moduser32 := syscall.NewLazyDLL("user32.dll")
	procOpenClipboard := moduser32.NewProc("OpenClipboard")
	procGetClipboardData := moduser32.NewProc("GetClipboardData")
	procCloseClipboard := moduser32.NewProc("CloseClipboard")

	// 打开剪贴板
	ret, _, _ := procOpenClipboard.Call(0)
	if ret == 0 {
		return "", adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("CLIPBOARD_OPEN_FAILED"),
			Error:      "Failed to open clipboard",
		}
	}
	defer procCloseClipboard.Call()

	// 获取剪贴板数据 (CF_UNICODETEXT = 13)
	hMem, _, _ := procGetClipboardData.Call(13)
	if hMem == 0 {
		return "", adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("CLIPBOARD_GET_FAILED"),
			Error:      "Failed to get clipboard data",
		}
	}

	// 锁定内存并读取文本
	modkernel32 := syscall.NewLazyDLL("kernel32.dll")
	procGlobalLock := modkernel32.NewProc("GlobalLock")
	procGlobalUnlock := modkernel32.NewProc("GlobalUnlock")

	ptr, _, _ := procGlobalLock.Call(hMem)
	if ptr == 0 {
		return "", adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("MEMORY_LOCK_FAILED"),
			Error:      "Failed to lock global memory",
		}
	}
	defer procGlobalUnlock.Call(hMem)

	// 读取 UTF-16 文本
	// 找到 null 终止符
	var textLen int
	for i := 0; ; i++ {
		if *(*uint16)(unsafe.Pointer(uintptr(ptr) + uintptr(i*2))) == 0 {
			textLen = i
			break
		}
	}

	// 转换为 Go 字符串
	utf16Slice := (*[1 << 20]uint16)(unsafe.Pointer(ptr))[:textLen:textLen]
	text := syscall.UTF16ToString(utf16Slice)

	return text, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

// Release 释放资源
func (b *Bridge) Release() {
	if b.initialized {
		procCoUninitialize.Call()
		b.initialized = false
	}
}

// EnumWindowsCallback 是 EnumWindows 的回调函数类型
// 返回非零值继续枚举，返回零值停止枚举
type EnumWindowsCallback func(hwnd uintptr, lParam uintptr) uintptr

// enumWindowsProcWrapper 是 EnumWindowsProc 的 Go 包装器
// 这是一个全局变量，用于存储当前的回调函数
var enumWindowsProcWrapper EnumWindowsCallback

// enumWindowsProc 是 C 语言风格的回调函数，必须使用 syscall.NewCallback 创建
func enumWindowsProc(hwnd uintptr, lParam uintptr) uintptr {
	if enumWindowsProcWrapper != nil {
		return enumWindowsProcWrapper(hwnd, lParam)
	}
	return 1 // 继续枚举
}

// enumerateWindows 使用 EnumWindows API 枚举所有顶级窗口
func enumerateWindows(callback EnumWindowsCallback) error {
	enumWindowsProcWrapper = callback
	defer func() { enumWindowsProcWrapper = nil }()

	procCallback := syscall.NewCallback(enumWindowsProc)
	ret, _, _ := procEnumWindows.Call(procCallback, 0)
	if ret == 0 {
		return syscall.GetLastError()
	}
	return nil
}

// EnumChildWindowsCallback 是 EnumChildWindows 的回调函数类型
// 返回非零值继续枚举，返回零值停止枚举
type EnumChildWindowsCallback func(hwnd uintptr, lParam uintptr) uintptr

// enumChildWindowsProcWrapper 是 EnumChildWindowsProc 的 Go 包装器
var enumChildWindowsProcWrapper EnumChildWindowsCallback

// enumChildWindowsProc 是 C 语言风格的回调函数，必须使用 syscall.NewCallback 创建
func enumChildWindowsProc(hwnd uintptr, lParam uintptr) uintptr {
	if enumChildWindowsProcWrapper != nil {
		return enumChildWindowsProcWrapper(hwnd, lParam)
	}
	return 1 // 继续枚举
}

// enumerateChildWindows 使用 EnumChildWindows API 枚举子窗口
func enumerateChildWindows(parentHandle uintptr, callback EnumChildWindowsCallback) error {
	enumChildWindowsProcWrapper = callback
	defer func() { enumChildWindowsProcWrapper = nil }()

	procCallback := syscall.NewCallback(enumChildWindowsProc)
	ret, _, _ := procEnumChildWindows.Call(parentHandle, procCallback, 0)
	if ret == 0 {
		return syscall.GetLastError()
	}
	return nil
}

// flattenNodeChildren 递归扁平化节点的子节点
func flattenNodeChildren(node AccessibleNode) []AccessibleNode {
	result := []AccessibleNode{}
	for _, child := range node.Children {
		result = append(result, child)
		// 递归获取子节点的子节点
		result = append(result, flattenNodeChildren(child)...)
	}
	return result
}

// IID_IAccessible 接口 ID
var IID_IAccessible = syscall.GUID{
	Data1: 0x618736E0,
	Data2: 0x3C3D,
	Data3: 0x11CF,
	Data4: [8]byte{0x81, 0x0C, 0x00, 0xAA, 0x00, 0x38, 0x9B, 0x71},
}
