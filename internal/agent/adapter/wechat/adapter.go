package wechat

import (
	"strconv"
	"time"

	"github.com/yourorg/auto-customer-service/internal/agent/adapter"
	"github.com/yourorg/auto-customer-service/internal/agent/windows"
	"github.com/yourorg/auto-customer-service/pkg/protocol"
)

// WeChatAdapter 微信桌面版适配器
type WeChatAdapter struct {
	config adapter.Config
	bridge windows.BridgeInterface
}

// NewWeChatAdapter 创建微信适配器实例
func NewWeChatAdapter() *WeChatAdapter {
	return &WeChatAdapter{
		bridge: windows.NewBridge(),
	}
}

// NewWeChatAdapterWithBridge 创建微信适配器实例（带依赖注入）
func NewWeChatAdapterWithBridge(bridge windows.BridgeInterface) *WeChatAdapter {
	return &WeChatAdapter{
		bridge: bridge,
	}
}

// Name 返回适配器名称
func (a *WeChatAdapter) Name() string {
	return "wechat"
}

// Version 返回适配器版本
func (a *WeChatAdapter) Version() string {
	return "1.0.0"
}

// SupportedApps 返回支持的应用列表
func (a *WeChatAdapter) SupportedApps() []string {
	return []string{"wechat"}
}

// Init 初始化适配器
func (a *WeChatAdapter) Init(config adapter.Config) adapter.Result {
	a.config = config

	// 初始化 Windows bridge
	if a.bridge != nil {
		result := a.bridge.Initialize()
		if result.Status != adapter.StatusSuccess {
			return adapter.Result{
				Status:     adapter.StatusFailed,
				ReasonCode: adapter.ReasonCode("BRIDGE_INIT_FAILED"),
				Error:      result.Error,
			}
		}
	}

	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
		ElapsedMs:  0,
	}
}

// Destroy 销毁适配器
func (a *WeChatAdapter) Destroy() adapter.Result {
	// 释放 bridge 资源
	if a.bridge != nil {
		a.bridge.Release()
	}

	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
		ElapsedMs:  0,
	}
}

// IsAvailable 检查适配器是否可用
func (a *WeChatAdapter) IsAvailable() adapter.Result {
	// 检查 bridge 是否可用
	if a.bridge == nil {
		return adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("BRIDGE_NOT_AVAILABLE"),
			Error:      "Bridge is not initialized",
		}
	}

	// 尝试初始化 bridge（如果尚未初始化）
	result := a.bridge.Initialize()
	if result.Status != adapter.StatusSuccess {
		return adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("BRIDGE_INIT_FAILED"),
			Error:      result.Error,
		}
	}

	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
		Confidence: 1.0,
		ElapsedMs:  0,
	}
}

// Detect 检测微信实例
func (a *WeChatAdapter) Detect() ([]protocol.AppInstanceRef, adapter.Result) {
	// 尝试多种方式查找微信窗口
	// 1. 按标题查找（中文微信）
	handles, result := a.bridge.FindTopLevelWindows("", "微信")
	if result.Status != adapter.StatusSuccess {
		// 如果按标题查找失败，尝试按类名查找
		handles, result = a.bridge.FindTopLevelWindows("WeChatMainWndForPC", "")
		if result.Status != adapter.StatusSuccess {
			return nil, adapter.Result{
				Status:     adapter.StatusFailed,
				ReasonCode: adapter.ReasonCode("WINDOW_NOT_FOUND"),
				Error:      "No WeChat window found",
			}
		}
	}

	instances := []protocol.AppInstanceRef{}
	for _, handle := range handles {
		// 获取窗口信息
		info, infoResult := a.bridge.GetWindowInfo(handle)
		if infoResult.Status == adapter.StatusSuccess {
			// 验证窗口类名是否为微信相关
			class, classResult := a.bridge.GetWindowClass(handle)
			isWeChatWindow := false
			if classResult.Status == adapter.StatusSuccess {
				// 检查类名是否包含微信相关标识
				if class == "WeChatMainWndForPC" || class == "WeChatLoginWndForPC" {
					isWeChatWindow = true
				}
			}

			// 如果类名验证失败，仍基于标题判断
			if !isWeChatWindow && info.Title != "" {
				// 检查标题是否包含微信标识
				if info.Title == "微信" || info.Title == "WeChat" {
					isWeChatWindow = true
				}
			}

			if isWeChatWindow {
				instances = append(instances, protocol.AppInstanceRef{
					AppID:      "wechat",
					InstanceID: info.Title,
				})
			}
		}
	}

	if len(instances) == 0 {
		return []protocol.AppInstanceRef{}, adapter.Result{
			Status:     adapter.StatusSuccess,
			ReasonCode: adapter.ReasonCode("NO_WECHAT_WINDOW"),
			Confidence: 0.0,
			ElapsedMs:  0,
		}
	}

	return instances, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
		Confidence: 1.0,
		ElapsedMs:  0,
	}
}

// Scan 扫描会话列表
func (a *WeChatAdapter) Scan(instance protocol.AppInstanceRef) ([]protocol.ConversationRef, adapter.Result) {
	// 查找微信窗口句柄
	handles, result := a.bridge.FindTopLevelWindows("", "微信")
	if result.Status != adapter.StatusSuccess {
		// 尝试按类名查找
		handles, result = a.bridge.FindTopLevelWindows("WeChatMainWndForPC", "")
		if result.Status != adapter.StatusSuccess {
			return nil, adapter.Result{
				Status:     adapter.StatusFailed,
				ReasonCode: adapter.ReasonCode("WINDOW_NOT_FOUND"),
				Error:      "No WeChat window found for scan",
			}
		}
	}

	if len(handles) == 0 {
		return []protocol.ConversationRef{}, adapter.Result{
			Status:     adapter.StatusSuccess,
			ReasonCode: adapter.ReasonCode("NO_CONVERSATIONS"),
			Confidence: 0.0,
			ElapsedMs:  0,
		}
	}

	// 使用第一个窗口句柄
	windowHandle := handles[0]

	// 获取窗口信息用于诊断
	info, infoResult := a.bridge.GetWindowInfo(windowHandle)

	// 使用 bridge 枚举可访问节点
	nodes, nodeResult := a.bridge.EnumerateAccessibleNodes(windowHandle)

	// 转换可访问节点为会话引用
	conversations := []protocol.ConversationRef{}

	if nodeResult.Status == adapter.StatusSuccess {
		// 真实实现：从可访问节点提取会话
		for i, node := range nodes {
			// 过滤出会话项（基于角色或类名）
			if node.Role == "list item" || node.Role == "ListItem" {
				conversations = append(conversations, protocol.ConversationRef{
					HostWindowHandle: windowHandle,
					AppInstance:      instance,
					DisplayName:      node.Name,
					ListPosition:     i,
				})
			}
		}
	} else {
		// Placeholder 实现：当无法枚举可访问节点时
		// 返回最小占位结果，包含诊断信息
		conversations = append(conversations, protocol.ConversationRef{
			HostWindowHandle: windowHandle,
			AppInstance:      instance,
			DisplayName:      "Placeholder Conversation",
			ListPosition:     0,
		})
	}

	// 构建诊断信息
	diagnostics := map[string]string{
		"window_handle":    strconv.FormatUint(uint64(windowHandle), 10),
		"window_class":     "",
		"window_title":     "",
		"nodes_found":      strconv.Itoa(len(nodes)),
		"implementation":   "partial",
	}

	if infoResult.Status == adapter.StatusSuccess {
		diagnostics["window_class"] = info.Class
		diagnostics["window_title"] = info.Title
	}

	if nodeResult.Status != adapter.StatusSuccess {
		diagnostics["implementation"] = "placeholder"
		diagnostics["enumerate_error"] = string(nodeResult.ReasonCode)
	}

	return conversations, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
		Confidence: 1.0,
		ElapsedMs:  0,
		Diagnostics: []adapter.Diagnostic{
			{
				Timestamp: time.Now(),
				Level:     "info",
				Message:   "Scan completed",
				Context:   diagnostics,
			},
		},
	}
}

// Focus 聚焦到指定会话
func (a *WeChatAdapter) Focus(conv protocol.ConversationRef) adapter.Result {
	// 聚焦到微信窗口
	result := a.bridge.FocusWindow(conv.HostWindowHandle)
	if result.Status != adapter.StatusSuccess {
		return result
	}

	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
		Confidence: 1.0,
		ElapsedMs:  0,
	}
}

// Read 读取消息
func (a *WeChatAdapter) Read(conv protocol.ConversationRef, limit int) ([]protocol.MessageObs, adapter.Result) {
	// 截图窗口用于 OCR 识别（stub 实现）
	_, result := a.bridge.CaptureWindow(conv.HostWindowHandle)
	if result.Status != adapter.StatusSuccess {
		return nil, result
	}

	// TODO: 实现 OCR 文字识别
	// 当前返回空消息列表作为 stub
	messages := []protocol.MessageObs{}
	return messages, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
		Confidence: 1.0,
		ElapsedMs:  0,
	}
}

// Send 发送消息
func (a *WeChatAdapter) Send(conv protocol.ConversationRef, content string, taskID string) adapter.Result {
	// 聚焦到窗口
	focusResult := a.bridge.FocusWindow(conv.HostWindowHandle)
	if focusResult.Status != adapter.StatusSuccess {
		return focusResult
	}

	// 使用剪贴板发送消息（复制粘贴方式）
	// 1. 设置剪贴板文本
	setResult := a.bridge.SetClipboardText(content)
	if setResult.Status != adapter.StatusSuccess {
		return setResult
	}

	// 2. 粘贴（Ctrl+V）
	sendResult := a.bridge.SendKeys(conv.HostWindowHandle, "^v")
	if sendResult.Status != adapter.StatusSuccess {
		return sendResult
	}

	// 3. 发送（Enter）
	sendResult = a.bridge.SendKeys(conv.HostWindowHandle, "{ENTER}")
	if sendResult.Status != adapter.StatusSuccess {
		return sendResult
	}

	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
		Confidence: 1.0,
		ElapsedMs:  0,
	}
}

// Verify 验证消息发送
func (a *WeChatAdapter) Verify(conv protocol.ConversationRef, content string, timeout time.Duration) (*protocol.MessageObs, adapter.Result) {
	// TODO: 验证消息是否成功发送
	// 当前返回成功作为 stub
	return nil, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
		Confidence: 1.0,
		ElapsedMs:  0,
	}
}

// CaptureDiagnostics 捕获诊断信息
func (a *WeChatAdapter) CaptureDiagnostics() (map[string]string, adapter.Result) {
	diagnostics := map[string]string{
		"adapter_name":    a.Name(),
		"adapter_version": a.Version(),
		"bridge_status":   "initialized",
	}

	// 尝试获取 bridge 诊断信息
	if a.bridge != nil {
		// 检查 bridge 是否已初始化
		// 通过尝试查找窗口来测试 bridge 功能
		_, result := a.bridge.FindTopLevelWindows("", "微信")
		if result.Status == adapter.StatusSuccess {
			diagnostics["bridge_status"] = "available"
		} else {
			diagnostics["bridge_status"] = "unavailable"
			diagnostics["bridge_error"] = result.Error
		}
	} else {
		diagnostics["bridge_status"] = "nil"
	}

	return diagnostics, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
		ElapsedMs:  0,
	}
}
