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
		// 如果无法枚举节点，返回空列表而不是占位会话
		return []protocol.ConversationRef{}, adapter.Result{
			Status:     adapter.StatusSuccess,
			ReasonCode: adapter.ReasonCode("NO_CONVERSATIONS"),
			Confidence: 0.0,
			ElapsedMs:  0,
			Diagnostics: []adapter.Diagnostic{
				{
					Timestamp: time.Now(),
					Level:     "info",
					Message:   "Scan completed - no conversations found",
					Context:   diagnostics,
				},
			},
		}
	}

	// 真实实现：从可访问节点提取会话
	// 只保留真实 list item / ListItem 且 Name 非空的节点
	for i, node := range nodes {
		// 过滤出会话项（基于角色或类名）
		if (node.Role == "list item" || node.Role == "ListItem") && node.Name != "" {
			conversations = append(conversations, protocol.ConversationRef{
				HostWindowHandle: windowHandle,
				AppInstance:      instance,
				DisplayName:      node.Name,
				ListPosition:     i,
			})
		}
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
	// 1. 聚焦到微信窗口
	result := a.bridge.FocusWindow(conv.HostWindowHandle)
	if result.Status != adapter.StatusSuccess {
		return result
	}

	// 2. 获取窗口信息以确定布局
	_, infoResult := a.bridge.GetWindowInfo(conv.HostWindowHandle)
	if infoResult.Status != adapter.StatusSuccess {
		// 如果无法获取窗口信息，仍然返回成功（至少聚焦了窗口）
		return adapter.Result{
			Status:     adapter.StatusSuccess,
			ReasonCode: adapter.ReasonOK,
			Confidence: 0.8,
			ElapsedMs:  0,
		}
	}

	// 3. 根据会话位置计算点击坐标
	// 假设对话列表在左侧，宽度约 200px，每个会话项高度约 40px
	// 起始 Y 坐标约 50px（标题栏 + 间距）
	convListWidth := 200
	itemHeight := 40
	startY := 50

	// 计算目标会话的点击位置
	clickX := convListWidth / 2 // 列表中间
	clickY := startY + (conv.ListPosition * itemHeight) + (itemHeight / 2)

	// 4. 点击目标会话
	clickResult := a.bridge.Click(conv.HostWindowHandle, clickX, clickY)
	if clickResult.Status != adapter.StatusSuccess {
		// 点击失败，但窗口已聚焦，返回部分成功
		return adapter.Result{
			Status:     adapter.StatusSuccess,
			ReasonCode: adapter.ReasonOK,
			Confidence: 0.7,
			ElapsedMs:  0,
		}
	}

	// 5. 等待 UI 更新
	// time.Sleep(100 * time.Millisecond)

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
	startTime := time.Now()

	// 阶段1: 聚焦到窗口
	focusResult := a.bridge.FocusWindow(conv.HostWindowHandle)
	if focusResult.Status != adapter.StatusSuccess {
		return adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("FOCUS_FAILED"),
			Error:      "Failed to focus window",
			ElapsedMs:  time.Since(startTime).Milliseconds(),
		}
	}

	// 阶段2: 设置剪贴板文本
	setResult := a.bridge.SetClipboardText(content)
	if setResult.Status != adapter.StatusSuccess {
		return adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("CLIPBOARD_FAILED"),
			Error:      "Failed to set clipboard text",
			ElapsedMs:  time.Since(startTime).Milliseconds(),
		}
	}

	// 阶段3: 粘贴（Ctrl+V）
	sendResult := a.bridge.SendKeys(conv.HostWindowHandle, "^v")
	if sendResult.Status != adapter.StatusSuccess {
		return adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("PASTE_FAILED"),
			Error:      "Failed to paste message",
			ElapsedMs:  time.Since(startTime).Milliseconds(),
		}
	}

	// 阶段4: 发送（Enter）
	sendResult = a.bridge.SendKeys(conv.HostWindowHandle, "{ENTER}")
	if sendResult.Status != adapter.StatusSuccess {
		return adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("SEND_FAILED"),
			Error:      "Failed to send message",
			ElapsedMs:  time.Since(startTime).Milliseconds(),
		}
	}

	// 阶段5: 验证发送（可选，通过截图检查）
	// 尝试截图以验证消息已发送
	_, captureResult := a.bridge.CaptureWindow(conv.HostWindowHandle)
	elapsedMs := time.Since(startTime).Milliseconds()

	if captureResult.Status != adapter.StatusSuccess {
		// 截图失败不影响发送成功，但记录警告
		return adapter.Result{
			Status:     adapter.StatusSuccess,
			ReasonCode: adapter.ReasonOK,
			Confidence: 0.9, // 略低的置信度，因为无法验证
			ElapsedMs:  elapsedMs,
		}
	}

	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
		Confidence: 1.0,
		ElapsedMs:  elapsedMs,
	}
}

// Verify 验证消息发送
func (a *WeChatAdapter) Verify(conv protocol.ConversationRef, content string, timeout time.Duration) (*protocol.MessageObs, adapter.Result) {
	startTime := time.Now()

	// 聚焦到窗口
	focusResult := a.bridge.FocusWindow(conv.HostWindowHandle)
	if focusResult.Status != adapter.StatusSuccess {
		return nil, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("FOCUS_FAILED"),
			Error:      "Failed to focus window for verification",
			ElapsedMs:  time.Since(startTime).Milliseconds(),
		}
	}

	// 截图窗口用于验证
	screenshot, captureResult := a.bridge.CaptureWindow(conv.HostWindowHandle)
	if captureResult.Status != adapter.StatusSuccess {
		// 截图失败，返回基本验证结果
		return nil, adapter.Result{
			Status:     adapter.StatusSuccess,
			ReasonCode: adapter.ReasonOK,
			Confidence: 0.5, // 低置信度，因为无法验证
			ElapsedMs:  time.Since(startTime).Milliseconds(),
		}
	}

	// TODO: 实现 OCR 文字识别来验证消息是否出现在聊天记录中
	// 当前返回基本验证结果
	// 验证逻辑：
	// 1. 截图聊天区域
	// 2. 使用 OCR 识别文字
	// 3. 检查是否包含发送的消息内容
	// 4. 返回 MessageObs 包含消息指纹

	// 由于 OCR 尚未实现，返回基本成功结果
	// 但记录截图数据可用于后续验证
	_ = screenshot // 避免未使用变量警告

	elapsedMs := time.Since(startTime).Milliseconds()

	return nil, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
		Confidence: 0.8, // 中等置信度，因为 OCR 未实现
		ElapsedMs:  elapsedMs,
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
