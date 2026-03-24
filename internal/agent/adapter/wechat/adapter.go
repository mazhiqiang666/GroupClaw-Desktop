package wechat

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mazhiqiang666/GroupClaw-Desktop/internal/agent/adapter"
	"github.com/mazhiqiang666/GroupClaw-Desktop/internal/agent/windows"
	"github.com/mazhiqiang666/GroupClaw-Desktop/pkg/protocol"
)

// WeChatAdapter 微信桌面版适配器
type WeChatAdapter struct {
	config                 adapter.Config
	bridge                 windows.BridgeInterface
	pathSystem             *PathSystem
	evidenceCollector      *EvidenceCollector
	messageClassifier      *MessageClassifier
	// Rule modules
	sessionCandidateRules  *SessionCandidateRules
	positioningRules       *PositioningStrategyRules
	activationRules        *ActivationVerificationRules
	messageRules           *MessageVerificationRules
	deliveryRules          *DeliveryAssessmentRules
}

// NewWeChatAdapter 创建微信适配器实例
func NewWeChatAdapter() *WeChatAdapter {
	pathSystem := NewPathSystem()
	evidenceCollector := NewEvidenceCollector()
	messageClassifier := NewMessageClassifier()

	return &WeChatAdapter{
		bridge:                 windows.NewBridge(),
		pathSystem:             pathSystem,
		evidenceCollector:      evidenceCollector,
		messageClassifier:      messageClassifier,
		sessionCandidateRules:  NewSessionCandidateRules(),
		positioningRules:       NewPositioningStrategyRules(pathSystem),
		activationRules:        NewActivationVerificationRules(pathSystem, evidenceCollector),
		messageRules:           NewMessageVerificationRules(pathSystem, messageClassifier, evidenceCollector),
		deliveryRules:          NewDeliveryAssessmentRules(),
	}
}

// NewWeChatAdapterWithBridge 创建微信适配器实例（带依赖注入）
func NewWeChatAdapterWithBridge(bridge windows.BridgeInterface) *WeChatAdapter {
	pathSystem := NewPathSystem()
	evidenceCollector := NewEvidenceCollector()
	messageClassifier := NewMessageClassifier()

	return &WeChatAdapter{
		bridge:                 bridge,
		pathSystem:             pathSystem,
		evidenceCollector:      evidenceCollector,
		messageClassifier:      messageClassifier,
		sessionCandidateRules:  NewSessionCandidateRules(),
		positioningRules:       NewPositioningStrategyRules(pathSystem),
		activationRules:        NewActivationVerificationRules(pathSystem, evidenceCollector),
		messageRules:           NewMessageVerificationRules(pathSystem, messageClassifier, evidenceCollector),
		deliveryRules:          NewDeliveryAssessmentRules(),
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
			Diagnostics: []adapter.Diagnostic{
				{
					Timestamp: time.Now(),
					Level:     "info",
					Message:   "No WeChat window found",
					Context: map[string]string{
						"locate_source":        "unknown",
						"evidence_count":       "0",
						"new_message_nodes":    "0",
						"message_content_match": "false",
						"delivery_state":       "unknown",
						"confidence":           "0.00",
					},
				},
			},
		}
	}

	return instances, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
		Confidence: 1.0,
		ElapsedMs:  0,
		Diagnostics: []adapter.Diagnostic{
			{
				Timestamp: time.Now(),
				Level:     "info",
				Message:   "WeChat window detected",
				Context: map[string]string{
					"locate_source":        "unknown",
					"evidence_count":       "0",
					"new_message_nodes":    "0",
					"message_content_match": "false",
					"delivery_state":       "unknown",
					"confidence":           "1.00",
				},
			},
		},
	}
}

// Scan 扫描会话列表
func (a *WeChatAdapter) Scan(instance protocol.AppInstanceRef) ([]protocol.ConversationRef, adapter.Result) {
	startTime := time.Now()

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

	// 构建诊断信息（提前定义，供视觉和回退路径使用）
	diagnostics := map[string]string{
		"window_handle":        strconv.FormatUint(uint64(windowHandle), 10),
		"window_class":         "",
		"window_title":         "",
		"nodes_found":          "0",
		"candidates_found":     "0",
		"hits_found":           "0",
		// Whitelist fields (always present for consistency)
		"locate_source":        "unknown",
		"evidence_count":       "0",
		"new_message_nodes":    "0",
		"message_content_match": "false",
		"delivery_state":       "unknown",
		"confidence":           "0.00",
	}

	if infoResult.Status == adapter.StatusSuccess {
		diagnostics["window_class"] = info.Class
		diagnostics["window_title"] = info.Title
	}

	// 优先尝试视觉扫描路线
	visionResult, visionAdapterResult := a.bridge.DetectConversations(windowHandle)
	var visualConversations []protocol.ConversationRef
	if visionAdapterResult.Status == adapter.StatusSuccess && len(visionResult.ConversationRects) > 0 {
		// 视觉扫描成功，对候选项进行评分和排序
		type scoredRect struct {
			rect           windows.ConversationRect
			originalIndex  int
			score          int
		}
		scoredRects := make([]scoredRect, len(visionResult.ConversationRects))
		for i, rect := range visionResult.ConversationRects {
			scoredRects[i] = scoredRect{
				rect:          rect,
				originalIndex: i,
				score:         scoreVisualCandidate(rect),
			}
		}
		// 按分数降序排序（分数高的优先）
		sort.Slice(scoredRects, func(i, j int) bool {
			return scoredRects[i].score > scoredRects[j].score
		})
		// 转换为ConversationRef，按排序后的顺序展示
		for rank, scored := range scoredRects {
			rect := scored.rect
			originalIndex := scored.originalIndex
			displayName := fmt.Sprintf("conversation_%d", rank) // 使用排序后的排名作为显示名
			previewText := fmt.Sprintf("rect:%d_%d_%d_%d", rect.X, rect.Y, rect.Width, rect.Height)
			hints := []string{
				fmt.Sprintf("visual_index:%d", originalIndex), // 原始视觉索引
				fmt.Sprintf("rect:%d_%d_%d_%d", rect.X, rect.Y, rect.Width, rect.Height),
				fmt.Sprintf("visual_rank:%d", rank),           // 排序后的排名
				fmt.Sprintf("original_visual_index:%d", originalIndex),
				fmt.Sprintf("visual_score:%d", scored.score),
			}
			if rect.HasAvatar {
				hints = append(hints, "has_avatar")
			}
			if rect.HasText {
				hints = append(hints, "has_text")
			}
			if rect.HasUnreadDot {
				hints = append(hints, "has_unread_dot")
			}
			if rect.IsSelected {
				hints = append(hints, "is_selected")
			}
			convRef := protocol.ConversationRef{
				HostWindowHandle: windowHandle,
				AppInstance:      instance,
				DisplayName:      displayName,
				ListPosition:     originalIndex, // 使用原始视觉索引作为ListPosition，供FocusConversationByVision使用
				PreviewText:      previewText,
				ListNeighborhoodHint: hints,
			}
			visualConversations = append(visualConversations, convRef)
		}
		// 更新诊断信息
		diagnostics["locate_source"] = "vision_scan"
		diagnostics["visual_conversations_found"] = strconv.Itoa(len(visualConversations))
		diagnostics["visual_rects_found"] = strconv.Itoa(len(visionResult.ConversationRects))
		diagnostics["visual_scan_status"] = "success"
		// 返回视觉扫描结果
		elapsedMs := time.Since(startTime).Milliseconds()
		return visualConversations, adapter.Result{
			Status:     adapter.StatusSuccess,
			ReasonCode: adapter.ReasonOK,
			Confidence: 1.0,
			ElapsedMs:  elapsedMs,
			Diagnostics: []adapter.Diagnostic{
				{
					Timestamp: time.Now(),
					Level:     "info",
					Message:   "Scan completed via vision detection",
					Context:   diagnostics,
				},
			},
		}
	}
	// 视觉扫描失败或未找到矩形，回退到旧的可访问性节点路线
	diagnostics["visual_scan_status"] = "failed_or_no_rects"
	if visionAdapterResult.Status != adapter.StatusSuccess {
		diagnostics["visual_scan_error"] = visionAdapterResult.Error
	}

	// 使用 bridge 枚举可访问节点
	nodes, nodeResult := a.bridge.EnumerateAccessibleNodes(windowHandle)

	// 更新诊断信息
	diagnostics["nodes_found"] = strconv.Itoa(len(nodes))
	// 其他字段在视觉路径失败时已设置初始值

	if infoResult.Status == adapter.StatusSuccess {
		diagnostics["window_class"] = info.Class
		diagnostics["window_title"] = info.Title
	}

	if nodeResult.Status != adapter.StatusSuccess {
		return []protocol.ConversationRef{}, adapter.Result{
			Status:     adapter.StatusSuccess,
			ReasonCode: adapter.ReasonCode("NO_CONVERSATIONS"),
			Confidence: 0.0,
			ElapsedMs:  0,
			Diagnostics: []adapter.Diagnostic{
				{
					Timestamp: time.Now(),
					Level:     "info",
					Message:   "Scan completed - node enumeration failed",
					Context:   diagnostics,
				},
			},
		}
	}

	// 递归扁平化整个节点树并生成路径
	flatNodes := a.pathSystem.FlattenNodesWithPath(nodes, "", 0, 10)

	// 获取窗口宽度用于 bounds 检查
	windowWidth := GetWindowWidthFromNodes(flatNodes)
	// 添加窗口宽度到诊断信息
	diagnostics["window_width"] = strconv.Itoa(windowWidth)
	diagnostics["flat_nodes_count"] = strconv.Itoa(len(flatNodes))

	// 使用规则模块筛选候选会话
	candidates := a.sessionCandidateRules.FilterCandidateConversations(flatNodes, windowWidth)

	// 如果候选为空，使用宽松的过滤器作为回退
	if len(candidates) == 0 && len(flatNodes) > 0 {
		diagnostics["fallback_used"] = "true"
		// 简单的宽松过滤器：有名称且边界有效
		for _, node := range flatNodes {
			if node.Name != "" && len(node.Bounds) == 4 {
				bounds := node.Bounds
				if bounds[0] >= -50 && bounds[1] >= -50 && bounds[2] > 0 && bounds[3] > 0 {
					candidates = append(candidates, node)
				}
			}
		}
	}

	// 转换为会话引用
	conversations := []protocol.ConversationRef{}
	for i, node := range candidates {
		// 生成父上下文和路径
		parentContext := generateParentContext(node)
		treePath := node.TreePath
		if treePath == "" {
			treePath = fmt.Sprintf("[%d]", i)
		}

		// 生成稳定定位信息
		stableKey := generateStableKey(node, parentContext, treePath)
		nodePath := treePath

		// 添加到会话列表
		conversations = append(conversations, protocol.ConversationRef{
			HostWindowHandle: windowHandle,
			AppInstance:      instance,
			DisplayName:      node.Name,
			ListPosition:     i,
			PreviewText:      stableKey,
			ListNeighborhoodHint: []string{
				nodePath,
				fmt.Sprintf("bounds:%d_%d_%d_%d", node.Bounds[0], node.Bounds[1], node.Bounds[2], node.Bounds[3]),
			},
		})
	}

	diagnostics["candidates_found"] = strconv.Itoa(len(candidates))
	diagnostics["hits_found"] = strconv.Itoa(len(conversations))

	elapsedMs := time.Since(startTime).Milliseconds()

	return conversations, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
		Confidence: 1.0,
		ElapsedMs:  elapsedMs,
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
	startTime := time.Now()

	// 1. 聚焦到微信窗口
	focusWindowResult := a.bridge.FocusWindow(conv.HostWindowHandle)
	if focusWindowResult.Status != adapter.StatusSuccess {
		return focusWindowResult
	}

	// 2. 优先尝试视觉Focus路线
	// 根据候选特征自适应选择点击策略
	strategy := "rect_center" // 默认策略
	// 检查候选项是否有文本（has_text）
	for _, hint := range conv.ListNeighborhoodHint {
		if hint == "has_text" {
			strategy = "text_center" // 如果候选项有文本，优先使用文本中心
			break
		}
	}
	visionResult, visionAdapterResult := a.bridge.FocusConversationByVision(
		conv.HostWindowHandle,
		strategy, // 自适应策略
		conv.ListPosition, // 使用ListPosition作为目标索引
		800, // 点击后等待800ms
	)

	// 如果视觉Focus成功（状态成功且focus_succeeded为true），直接返回
	if visionAdapterResult.Status == adapter.StatusSuccess && visionResult.FocusSucceeded {
		// 将VisionFocusResult映射到adapter.Result
		return a.convertVisionFocusResult(visionResult, startTime, "vision")
	}

	// 3. 视觉Focus失败或未完全成功，回退到旧的可访问性节点路线
	// 记录视觉Focus尝试的诊断信息
	visionDiagLevel := "info"
	if visionAdapterResult.Status != adapter.StatusSuccess {
		visionDiagLevel = "warn"
	} else if !visionResult.FocusSucceeded {
		visionDiagLevel = "warn"
	}

	visionDiagMessage := "Vision focus attempted"
	if visionAdapterResult.Status != adapter.StatusSuccess {
		visionDiagMessage = fmt.Sprintf("Vision focus failed with status: %s", visionAdapterResult.Status)
	} else if !visionResult.FocusSucceeded {
		visionDiagMessage = fmt.Sprintf("Vision focus verification failed (confidence: %.2f)", visionResult.FocusConfidence)
	}

	// 继续原有逻辑...

	// 2. 重新枚举节点以找到目标会话的 bounds
	nodes, nodeResult := a.bridge.EnumerateAccessibleNodes(conv.HostWindowHandle)
	if nodeResult.Status != adapter.StatusSuccess {
		// 无法枚举节点，使用 ListPosition 回退方案
		return a.focusByListPosition(conv, startTime, "fallback_enumerate_failed")
	}

	// 递归扁平化节点树并生成路径
	flatNodes := a.pathSystem.FlattenNodesWithPath(nodes, "", 0, 10)

	// 3. 使用规则模块查找匹配的节点
	positioningResult := a.positioningRules.FindNodeByStrategy(flatNodes, conv)

	var targetNode *windows.AccessibleNode
	var locateSource string

	if positioningResult.Node != nil {
		targetNode = positioningResult.Node
		locateSource = positioningResult.Source
	}

	// 4. 如果未找到节点，使用 ListPosition 回退
	if targetNode == nil {
		return a.focusByListPosition(conv, startTime, "fallback_node_not_found")
	}

	// 5. 根据节点 bounds 计算点击位置
	clickX, clickY, ok := a.positioningRules.CalculateClickPosition(targetNode)
	if !ok {
		return a.focusByListPosition(conv, startTime, "fallback_no_bounds")
	}

	// 6. 点击目标会话
	clickResult := a.bridge.Click(conv.HostWindowHandle, clickX, clickY)
	if clickResult.Status != adapter.StatusSuccess {
		// 点击失败，但窗口已聚焦，返回部分成功
		return adapter.Result{
			Status:     adapter.StatusSuccess,
			ReasonCode: adapter.ReasonOK,
			Confidence: 0.7,
			ElapsedMs:  time.Since(startTime).Milliseconds(),
			Diagnostics: []adapter.Diagnostic{
				{
					Timestamp: time.Now(),
					Level:     "warn",
					Message:   "Focus click failed but window focused",
					Context: map[string]string{
						"locate_source": locateSource,
						"click_x":       strconv.Itoa(clickX),
						"click_y":       strconv.Itoa(clickY),
					},
				},
			},
		}
	}

	// 7. 等待 UI 更新
	time.Sleep(100 * time.Millisecond)

	// 8. 验证会话是否已激活（使用规则模块）
	evidence := a.activationRules.VerifySessionActivation(conv, nodes, []windows.AccessibleNode{}, locateSource)

	elapsedMs := time.Since(startTime).Milliseconds()

	// 9. 根据验证结果返回
	assessment := a.deliveryRules.AssessFocusOnlyState(evidence)

	diagnostics := ConvertFocusEvidenceToDiagnostics(evidence)
	// Add non-applicable whitelist fields with default values
	diagnostics["new_message_nodes"] = "0"
	diagnostics["message_content_match"] = "false"
	diagnostics["delivery_state"] = "unknown"
	diagnostics["click_x"] = strconv.Itoa(clickX)
	diagnostics["click_y"] = strconv.Itoa(clickY)
	diagnostics["elapsed_ms"] = strconv.FormatInt(elapsedMs, 10)

	// 添加视觉尝试的诊断信息
	visionAttemptDiagnostic := adapter.Diagnostic{
		Timestamp: time.Now(),
		Level:     visionDiagLevel,
		Message:   visionDiagMessage,
		Context: map[string]string{
			"vision_focus_attempted": "true",
			"vision_focus_status":    string(visionAdapterResult.Status),
			"vision_focus_succeeded": strconv.FormatBool(visionResult.FocusSucceeded),
			"vision_focus_confidence": fmt.Sprintf("%.2f", visionResult.FocusConfidence),
			"vision_click_strategy":   visionResult.ClickStrategy,
			"vision_target_index":     strconv.Itoa(conv.ListPosition),
		},
	}

	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
		Confidence: assessment.Confidence,
		ElapsedMs:  elapsedMs,
		Diagnostics: []adapter.Diagnostic{
			visionAttemptDiagnostic,
			{
				Timestamp: time.Now(),
				Level:     "info",
				Message:   "Focus completed with verification (fallback to accessibility)",
				Context:   diagnostics,
			},
		},
	}
}

// focusByListPosition 使用 ListPosition 回退方案计算点击位置
func (a *WeChatAdapter) focusByListPosition(conv protocol.ConversationRef, startTime time.Time, reason string) adapter.Result {
	// 获取窗口信息以确定布局
	_, infoResult := a.bridge.GetWindowInfo(conv.HostWindowHandle)
	if infoResult.Status != adapter.StatusSuccess {
		// 如果无法获取窗口信息，仍然返回成功（至少聚焦了窗口）
		return adapter.Result{
			Status:     adapter.StatusSuccess,
			ReasonCode: adapter.ReasonOK,
			Confidence: 0.8,
			ElapsedMs:  time.Since(startTime).Milliseconds(),
			Diagnostics: []adapter.Diagnostic{
				{
					Timestamp: time.Now(),
					Level:     "warn",
					Message:   "Focus using fallback - cannot get window info",
					Context: map[string]string{
						"reason": reason,
					},
				},
			},
		}
	}

	// 根据会话位置计算点击坐标
	// 假设对话列表在左侧，宽度约 200px，每个会话项高度约 40px
	// 起始 Y 坐标约 50px（标题栏 + 间距）
	convListWidth := 200
	itemHeight := 40
	startY := 50

	// 计算目标会话的点击位置
	clickX := convListWidth / 2 // 列表中间
	clickY := startY + (conv.ListPosition * itemHeight) + (itemHeight / 2)

	// 点击目标会话
	clickResult := a.bridge.Click(conv.HostWindowHandle, clickX, clickY)
	if clickResult.Status != adapter.StatusSuccess {
		// 点击失败，但窗口已聚焦，返回部分成功
		return adapter.Result{
			Status:     adapter.StatusSuccess,
			ReasonCode: adapter.ReasonOK,
			Confidence: 0.7,
			ElapsedMs:  time.Since(startTime).Milliseconds(),
			Diagnostics: []adapter.Diagnostic{
				{
					Timestamp: time.Now(),
					Level:     "warn",
					Message:   "Focus click failed using fallback",
					Context: map[string]string{
						"reason":    reason,
						"click_x":   strconv.Itoa(clickX),
						"click_y":   strconv.Itoa(clickY),
					},
				},
			},
		}
	}

	// 等待 UI 更新
	time.Sleep(100 * time.Millisecond)

	elapsedMs := time.Since(startTime).Milliseconds()

	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
		Confidence: 0.9, // 略低置信度，因为使用的是回退方案
		ElapsedMs:  elapsedMs,
		Diagnostics: []adapter.Diagnostic{
			{
				Timestamp: time.Now(),
				Level:     "info",
				Message:   "Focus completed using fallback",
				Context: map[string]string{
					"locate_source":        "list_position",
					"reason":               reason,
					"click_x":              strconv.Itoa(clickX),
					"click_y":              strconv.Itoa(clickY),
					"elapsed_ms":           strconv.FormatInt(elapsedMs, 10),
					// Whitelist fields (always present for consistency)
					"evidence_count":       "0",
					"new_message_nodes":    "0",
					"message_content_match": "false",
					"delivery_state":       "unknown",
					"confidence":           "0.90",
				},
			},
		},
	}
}

// Read 读取消息
func (a *WeChatAdapter) Read(conv protocol.ConversationRef, limit int) ([]protocol.MessageObs, adapter.Result) {
	startTime := time.Now()

	// 枚举可访问节点
	nodes, nodeResult := a.bridge.EnumerateAccessibleNodes(conv.HostWindowHandle)
	if nodeResult.Status != adapter.StatusSuccess {
		return nil, adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("NODE_ENUM_FAILED"),
			Error:      "Failed to enumerate accessible nodes",
			ElapsedMs:  time.Since(startTime).Milliseconds(),
			Diagnostics: []adapter.Diagnostic{
				{
					Timestamp: time.Now(),
					Level:     "error",
					Message:   "Node enumeration failed in Read",
					Context: map[string]string{
						"host_window_handle": strconv.FormatUint(uint64(conv.HostWindowHandle), 10),
						"locate_source":      "read",
						"evidence_count":     "0",
					},
				},
			},
		}
	}

	// 扁平化节点树
	flatNodes := a.pathSystem.FlattenNodesWithPath(nodes, "", 0, 10)

	// 过滤消息区域节点
	messageNodes := a.messageClassifier.FilterMessageAreaNodes(flatNodes, conv.HostWindowHandle)

	// 转换为消息观察结果
	messages := []protocol.MessageObs{}
	for i, node := range messageNodes {
		if limit > 0 && len(messages) >= limit {
			break
		}
		// 创建基础消息观察结果
		msg := protocol.MessageObs{
			MessageID:          fmt.Sprintf("read_%d_%d", conv.HostWindowHandle, i),
			ConversationID:     conv.DisplayName,
			SenderSide:         "unknown",
			NormalizedText:     node.Name,
			Timestamp:          time.Now(),
			ObservedAt:         time.Now(),
			MessageFingerprint: fmt.Sprintf("node_%s_%s", node.TreePath, node.Name),
			NeighborFingerprint: fmt.Sprintf("bounds_%d_%d_%d_%d", node.Bounds[0], node.Bounds[1], node.Bounds[2], node.Bounds[3]),
		}
		messages = append(messages, msg)
	}

	elapsedMs := time.Since(startTime).Milliseconds()

	if len(messages) == 0 {
		return messages, adapter.Result{
			Status:     adapter.StatusSuccess,
			ReasonCode: adapter.ReasonCode("NO_MESSAGES_FOUND"),
			Confidence: 0.0,
			ElapsedMs:  elapsedMs,
			Diagnostics: []adapter.Diagnostic{
				{
					Timestamp: time.Now(),
					Level:     "info",
					Message:   "No message nodes found in conversation",
					Context: map[string]string{
						"host_window_handle": strconv.FormatUint(uint64(conv.HostWindowHandle), 10),
						"flat_nodes_count":   strconv.Itoa(len(flatNodes)),
						"message_nodes_found": strconv.Itoa(len(messageNodes)),
						"locate_source":      "read",
						"evidence_count":     "0",
					},
				},
			},
		}
	}

	return messages, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
		Confidence: 0.7,
		ElapsedMs:  elapsedMs,
		Diagnostics: []adapter.Diagnostic{
			{
				Timestamp: time.Now(),
				Level:     "info",
				Message:   "Successfully read messages from accessible nodes",
				Context: map[string]string{
					"host_window_handle": strconv.FormatUint(uint64(conv.HostWindowHandle), 10),
					"flat_nodes_count":   strconv.Itoa(len(flatNodes)),
					"message_nodes_found": strconv.Itoa(len(messageNodes)),
					"messages_returned":   strconv.Itoa(len(messages)),
					"locate_source":      "read",
					"evidence_count":     strconv.Itoa(len(messageNodes)),
				},
			},
		},
	}
}


// Send 发送消息
func (a *WeChatAdapter) Send(conv protocol.ConversationRef, content string, taskID string) adapter.Result {
	startTime := time.Now()

	// 阶段1: 使用适配器的Focus方法聚焦到会话
	focusResult := a.Focus(conv)
	if focusResult.Status != adapter.StatusSuccess {
		return adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("FOCUS_FAILED"),
			Error:      "Failed to focus conversation",
			ElapsedMs:  time.Since(startTime).Milliseconds(),
			Diagnostics: focusResult.Diagnostics, // 传递Focus的诊断信息
		}
	}

	// 提取Focus诊断信息
	focusLocateSource := "unknown"
	focusConfidence := "0.00"
	focusClickStrategy := "unknown"
	sendAfterFocus := "true"

	// 从Focus结果中提取关键诊断信息
	for _, diag := range focusResult.Diagnostics {
		for k, v := range diag.Context {
			switch k {
			case "locate_source":
				focusLocateSource = v
			case "focus_confidence": // 优先读取视觉Focus返回的focus_confidence
				focusConfidence = v
			case "confidence": // 向后兼容旧路径的confidence字段
				if focusConfidence == "0.00" {
					focusConfidence = v
				}
			case "click_strategy":
				focusClickStrategy = v
			case "click_source":
				// 如果click_strategy为unknown，则使用click_source作为回退
				if focusClickStrategy == "unknown" {
					focusClickStrategy = v
				}
			}
		}
	}

	// 检查是否为输入框调试模式
	isInputBoxDebugMode := false
	if strings.Contains(taskID, "debug_input") || strings.Contains(taskID, "test_input") {
		isInputBoxDebugMode = true
	}

	// 如果Focus置信度过低（<0.5），可以决定降级处理
	focusConfidenceFloat := 0.0
	if conf, err := strconv.ParseFloat(focusConfidence, 64); err == nil {
		focusConfidenceFloat = conf
	}

	// 阶段2: 检测并点击输入框（解决发送失败问题）
	inputBoxClicked := false
	inputClickX := 0
	inputClickY := 0
	inputClickSource := "not_attempted"
	inputBoxClickAttempts := 0
	inputBoxClickSuccess := false
	var inputBoxRect windows.InputBoxRect
	windowWidth := 0
	windowHeight := 0
	inputBoxDiffAfterClick := 0.0  // 输入框点击后差异百分比

	// 截图变量用于增强验证
	var beforeClickScreenshot []byte    // 输入框点击前截图
	var afterClickScreenshot []byte     // 输入框点击后截图
	var afterPasteScreenshot []byte     // 粘贴后截图
	var afterEnter300msScreenshot []byte  // Enter后300ms截图
	var afterEnter800msScreenshot []byte  // Enter后800ms截图
	var afterEnter1500msScreenshot []byte // Enter后1500ms截图

	// 检测左侧边栏矩形（通过视觉检测会话列表）
	visionResult, visionDetectResult := a.bridge.DetectConversations(conv.HostWindowHandle)
	if visionDetectResult.Status == adapter.StatusSuccess {
		windowWidth = visionResult.WindowWidth
		windowHeight = visionResult.WindowHeight

		// 检测输入框区域（使用视觉检测到的窗口尺寸）
		detectedRect, inputBoxResult := a.bridge.DetectInputBoxArea(
			conv.HostWindowHandle,
			visionResult.LeftSidebarRect,
			visionResult.WindowWidth,
			visionResult.WindowHeight,
		)
		inputBoxRect = detectedRect

		if inputBoxResult.Status == adapter.StatusSuccess {
			// 1. 捕获输入框点击前截图
			beforeClickScreenshot, _ = a.bridge.CaptureWindow(conv.HostWindowHandle)

			// 定义输入框点击策略列表（按优先级排序）
			strategies := []string{"input_left_third", "input_center", "input_left_quarter", "input_double_click_center"}
			selectedStrategy := ""
			selectedClickX := 0
			selectedClickY := 0
			selectedClickSource := ""
			localClickAttempts := 0
			localClickSuccess := false
			// 使用外部定义的 inputBoxDiffAfterClick 变量

			// 遍历策略，直到找到能激活输入框的策略
			for _, strategy := range strategies {
				// 计算该策略的点击坐标
				clickX, clickY, clickSource := a.bridge.GetInputBoxClickPoint(inputBoxRect, strategy)
				maxClickAttempts := 2
				strategySuccess := false
				strategyAttempts := 0
				var strategyAfterClickScreenshot []byte

				for attempt := 1; attempt <= maxClickAttempts && !strategySuccess; attempt++ {
					clickResult := a.bridge.Click(conv.HostWindowHandle, clickX, clickY)
					if clickResult.Status == adapter.StatusSuccess {
						strategySuccess = true
						strategyAttempts = attempt
						// 等待点击生效
						time.Sleep(200 * time.Millisecond)
						// 捕获点击后截图
						strategyAfterClickScreenshot, _ = a.bridge.CaptureWindow(conv.HostWindowHandle)
					} else if attempt < maxClickAttempts {
						time.Sleep(100 * time.Millisecond)
					}
				}

				if strategySuccess && strategyAfterClickScreenshot != nil && beforeClickScreenshot != nil {
					// 计算输入框区域差异
					diff := windows.CalculateRectDiffPercent(beforeClickScreenshot, strategyAfterClickScreenshot, windowWidth, windowHeight, inputBoxRect)
					inputBoxDiffAfterClick = diff
					// 如果差异大于0，认为输入框被激活
					if diff > 0 {
						selectedStrategy = strategy
						selectedClickX = clickX
						selectedClickY = clickY
						selectedClickSource = clickSource
						localClickAttempts = strategyAttempts
						localClickSuccess = true
						afterClickScreenshot = strategyAfterClickScreenshot
						break // 找到有效策略，退出循环
					}
					// 如果差异为0，继续尝试下一个策略
				}
			}

			// 如果没有策略成功激活输入框，回退到第一个策略
			if !localClickSuccess && len(strategies) > 0 {
				fallbackStrategy := strategies[0]
				clickX, clickY, clickSource := a.bridge.GetInputBoxClickPoint(inputBoxRect, fallbackStrategy)
				// 尝试点击
				maxClickAttempts := 2
				for attempt := 1; attempt <= maxClickAttempts && !localClickSuccess; attempt++ {
					clickResult := a.bridge.Click(conv.HostWindowHandle, clickX, clickY)
					if clickResult.Status == adapter.StatusSuccess {
						localClickSuccess = true
						localClickAttempts = attempt
						selectedStrategy = fallbackStrategy
						selectedClickX = clickX
						selectedClickY = clickY
						selectedClickSource = clickSource
						time.Sleep(200 * time.Millisecond)
						afterClickScreenshot, _ = a.bridge.CaptureWindow(conv.HostWindowHandle)
						// 计算回退策略的差异
						if afterClickScreenshot != nil && beforeClickScreenshot != nil {
							diff := windows.CalculateRectDiffPercent(beforeClickScreenshot, afterClickScreenshot, windowWidth, windowHeight, inputBoxRect)
							inputBoxDiffAfterClick = diff
						}
					} else if attempt < maxClickAttempts {
						time.Sleep(100 * time.Millisecond)
					}
				}
			}

			// 设置输出变量
			inputBoxClicked = localClickSuccess
			inputClickX = selectedClickX
			inputClickY = selectedClickY
			inputClickSource = selectedClickSource
			if selectedStrategy != "" {
				inputClickSource = selectedClickSource + "_strategy_" + selectedStrategy
			}
			inputBoxClickAttempts = localClickAttempts
			inputBoxClickSuccess = localClickSuccess
		}
	}

	// 输入框调试模式：如果taskID包含debug_input或test_input，则只进行输入框点击测试
	if isInputBoxDebugMode {
		// 返回输入框点击测试结果，不执行后续发送流程
		return adapter.Result{
			Status:     adapter.StatusSuccess,
			ReasonCode: adapter.ReasonOK,
			ElapsedMs:  time.Since(startTime).Milliseconds(),
			Diagnostics: []adapter.Diagnostic{
				{
					Timestamp: time.Now(),
					Level:     "info",
					Message:   "Input box click test completed",
					Context: map[string]string{
						"input_box_clicked":           strconv.FormatBool(inputBoxClicked),
						"input_click_x":               strconv.Itoa(inputClickX),
						"input_click_y":               strconv.Itoa(inputClickY),
						"input_click_source":          inputClickSource,
						"input_box_click_attempts":    strconv.Itoa(inputBoxClickAttempts),
						"input_box_click_success":     strconv.FormatBool(inputBoxClickSuccess),
						"input_box_x":                 strconv.Itoa(inputBoxRect.X),
						"input_box_y":                 strconv.Itoa(inputBoxRect.Y),
						"input_box_width":             strconv.Itoa(inputBoxRect.Width),
						"input_box_height":            strconv.Itoa(inputBoxRect.Height),
						"input_box_diff_after_click":  fmt.Sprintf("%.3f", inputBoxDiffAfterClick),
						"window_width":                strconv.Itoa(windowWidth),
						"window_height":               strconv.Itoa(windowHeight),
						"debug_mode":                  "input_box_click_test",
						"message_skipped":             "true",
					},
				},
			},
		}
	}

	// 阶段3: 发送前捕获消息区域节点（用于后续差异比较）
	nodesBefore, nodesBeforeResult := a.bridge.EnumerateAccessibleNodes(conv.HostWindowHandle)
	if nodesBeforeResult.Status != adapter.StatusSuccess {
		nodesBefore = nil
	}

	// 阶段2b: 发送前截图（用于后续差异比较）
	beforeScreenshot, beforeResult := a.bridge.CaptureWindow(conv.HostWindowHandle)
	if beforeResult.Status != adapter.StatusSuccess {
		// 截图失败不影响发送，但记录警告
		beforeScreenshot = nil
	}

	// 阶段3: 设置剪贴板文本
	setResult := a.bridge.SetClipboardText(content)
	if setResult.Status != adapter.StatusSuccess {
		return adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("CLIPBOARD_FAILED"),
			Error:      "Failed to set clipboard text",
			ElapsedMs:  time.Since(startTime).Milliseconds(),
		}
	}

	// 阶段4: 粘贴（Ctrl+V）
	sendResult := a.bridge.SendKeys(conv.HostWindowHandle, "^v")
	if sendResult.Status != adapter.StatusSuccess {
		return adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("PASTE_FAILED"),
			Error:      "Failed to paste message",
			ElapsedMs:  time.Since(startTime).Milliseconds(),
		}
	}
	// 等待粘贴完成
	time.Sleep(50 * time.Millisecond)
	// 捕获粘贴后截图
	afterPasteScreenshot, _ = a.bridge.CaptureWindow(conv.HostWindowHandle)

	// 阶段5: 发送（Enter）
	sendResult = a.bridge.SendKeys(conv.HostWindowHandle, "{ENTER}")
	if sendResult.Status != adapter.StatusSuccess {
		return adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("SEND_FAILED"),
			Error:      "Failed to send message",
			ElapsedMs:  time.Since(startTime).Milliseconds(),
		}
	}

	// 捕获Enter后多时间点截图
	// 300ms后
	time.Sleep(300 * time.Millisecond)
	afterEnter300msScreenshot, _ = a.bridge.CaptureWindow(conv.HostWindowHandle)

	// 800ms后（再等500ms）
	time.Sleep(500 * time.Millisecond)
	afterEnter800msScreenshot, _ = a.bridge.CaptureWindow(conv.HostWindowHandle)

	// 1500ms后（再等700ms）
	time.Sleep(700 * time.Millisecond)
	afterEnter1500msScreenshot, _ = a.bridge.CaptureWindow(conv.HostWindowHandle)

	// 阶段6: 发送后捕获消息区域节点（用于差异比较）
	nodesAfter, nodesAfterResult := a.bridge.EnumerateAccessibleNodes(conv.HostWindowHandle)
	var messageNodesAfter []windows.AccessibleNode
	if nodesAfterResult.Status == adapter.StatusSuccess {
		flatNodes := a.pathSystem.FlattenNodesWithPath(nodesAfter, "", 0, 10)
		messageNodesAfter = a.messageClassifier.FilterMessageAreaNodes(flatNodes, conv.HostWindowHandle)
	}

	// 阶段6b: 发送后截图（用于差异比较） - 使用1500ms后的截图作为最终截图
	var afterScreenshot []byte
	afterScreenshot = afterEnter1500msScreenshot
	elapsedMs := time.Since(startTime).Milliseconds()

	// 阶段7: 使用规则模块验证消息发送
	chatAreaBounds := [4]int{}
	if len(messageNodesAfter) > 0 {
		chatAreaBounds = messageNodesAfter[0].Bounds
	}

	// 如果视觉检测失败，提供默认窗口尺寸
	if windowWidth == 0 || windowHeight == 0 {
		windowWidth = 800
		windowHeight = 600
	}

	var messageEvidence SendVerificationEvidence
	// 始终使用增强验证函数（即使某些参数为空，函数内部会处理）
	messageEvidence = a.messageRules.VerifyMessageSendEnhanced(
		nodesBefore, nodesAfter,
		beforeScreenshot, afterScreenshot,
		chatAreaBounds, content,
		inputBoxClicked,
		inputBoxRect,
		windowWidth, windowHeight,
		beforeClickScreenshot,
		afterClickScreenshot,
		afterPasteScreenshot,
		afterEnter300msScreenshot,
		afterEnter800msScreenshot,
		afterEnter1500msScreenshot,
	)

	// 阶段8: 生成最终评估
	assessment := a.deliveryRules.AssessDeliveryState(
		FocusVerificationEvidence{Confidence: 1.0}, // Focus was successful
		messageEvidence,
	)

	// 合并诊断信息
	diagnostics := make(map[string]string)
	// 添加消息证据诊断
	for k, v := range ConvertMessageEvidenceToDiagnostics(messageEvidence) {
		diagnostics[k] = v
	}
	// 添加交付评估诊断
	for k, v := range ConvertDeliveryAssessmentToDiagnostics(assessment) {
		diagnostics[k] = v
	}
	// 添加聚焦证据诊断（简化版）
	focusEvidence := FocusVerificationEvidence{
		Confidence:    1.0,
		LocateSource:  "send",
		EvidenceCount: messageEvidence.NewMessageNodes,
	}
	for k, v := range ConvertFocusEvidenceToDiagnostics(focusEvidence) {
		diagnostics[k] = v
	}
	// 添加Focus相关诊断字段
	diagnostics["focus_locate_source"] = focusLocateSource
	diagnostics["focus_confidence"] = focusConfidence
	diagnostics["focus_click_strategy"] = focusClickStrategy
	diagnostics["send_after_focus"] = sendAfterFocus

	// 根据Focus置信度决定是否降级处理
	if focusConfidenceFloat < 0.5 {
		diagnostics["focus_confidence_low"] = "true"
		// 可以考虑降级处理逻辑，但当前仅记录诊断
	}

	// 添加输入框点击诊断字段
	diagnostics["input_box_clicked"] = strconv.FormatBool(inputBoxClicked)
	diagnostics["input_click_x"] = strconv.Itoa(inputClickX)
	diagnostics["input_click_y"] = strconv.Itoa(inputClickY)
	diagnostics["input_click_source"] = inputClickSource
	diagnostics["input_box_click_attempts"] = strconv.Itoa(inputBoxClickAttempts)
	diagnostics["input_box_click_success"] = strconv.FormatBool(inputBoxClickSuccess)
	// 添加输入框矩形信息用于调试
	diagnostics["input_box_x"] = strconv.Itoa(inputBoxRect.X)
	diagnostics["input_box_y"] = strconv.Itoa(inputBoxRect.Y)
	diagnostics["input_box_width"] = strconv.Itoa(inputBoxRect.Width)
	diagnostics["input_box_height"] = strconv.Itoa(inputBoxRect.Height)
	// 添加输入框点击后差异信息
	diagnostics["input_box_diff_after_click_debug"] = fmt.Sprintf("%.3f", inputBoxDiffAfterClick)
	diagnostics["paste_executed"] = "true"  // 当前流程中必定执行粘贴
	diagnostics["enter_executed"] = "true"   // 当前流程中必定执行发送

	// 添加内容长度
	diagnostics["content_length"] = strconv.Itoa(len(content))

	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
		Confidence: assessment.Confidence,
		ElapsedMs:  elapsedMs,
		Diagnostics: []adapter.Diagnostic{
			{
				Timestamp: time.Now(),
				Level:     "info",
				Message:   "Send completed with verification",
				Context:   diagnostics,
			},
		},
	}
}

// Verify 验证消息发送（强验证）
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

	// 阶段1: 枚举可访问节点以检查消息区域变化
	nodes, nodeResult := a.bridge.EnumerateAccessibleNodes(conv.HostWindowHandle)

	var messageEvidence SendVerificationEvidence

	if nodeResult.Status == adapter.StatusSuccess {
		// 扁平化节点树并生成路径
		flatNodes := a.pathSystem.FlattenNodesWithPath(nodes, "", 0, 10)

		// 获取消息区域节点
		messageNodes := a.messageClassifier.FilterMessageAreaNodes(flatNodes, conv.HostWindowHandle)

		// 使用规则模块验证消息
		messageEvidence = a.messageRules.VerifyMessageSend(
			[]windows.AccessibleNode{}, // No before nodes for verify
			nodes,
			[]byte{}, // No before screenshot
			[]byte{}, // No after screenshot (would need to capture)
			[4]int{},
			content,
		)

		// 检查是否有包含发送内容的节点
		for _, node := range messageNodes {
			if node.Name != "" && content != "" && strings.Contains(node.Name, content) {
				messageEvidence.MessageContentMatch = true
				break
			}
		}
	}

	// 计算置信度和交付状态
	assessment := a.deliveryRules.AssessDeliveryState(
		FocusVerificationEvidence{Confidence: 1.0}, // Focus was successful
		messageEvidence,
	)

	elapsedMs := time.Since(startTime).Milliseconds()

	// 创建消息观察结果
	var verifiedMessage *protocol.MessageObs
	if assessment.Confidence >= 0.5 {
		// 构建验证通过的消息观察
		verifiedMessage = &protocol.MessageObs{
			MessageID:          fmt.Sprintf("verify_%d_%d", conv.HostWindowHandle, time.Now().UnixNano()),
			ConversationID:     conv.DisplayName,
			SenderSide:         "self",
			NormalizedText:     content,
			Timestamp:          time.Now(),
			ObservedAt:         time.Now(),
			MessageFingerprint: fmt.Sprintf("verify_%s_%s", conv.DisplayName, content),
			NeighborFingerprint: fmt.Sprintf("confidence_%.2f_nodes_%d", assessment.Confidence, messageEvidence.NewMessageNodes),
		}
	} else {
		// 验证置信度低，返回nil
		verifiedMessage = nil
	}

	// 合并诊断信息
	diagnostics := make(map[string]string)
	// 添加消息证据诊断
	for k, v := range ConvertMessageEvidenceToDiagnostics(messageEvidence) {
		diagnostics[k] = v
	}
	// 添加交付评估诊断
	for k, v := range ConvertDeliveryAssessmentToDiagnostics(assessment) {
		diagnostics[k] = v
	}
	// 添加聚焦证据诊断（简化版）
	focusEvidence := FocusVerificationEvidence{
		Confidence:    1.0,
		LocateSource:  "verify",
		EvidenceCount: messageEvidence.NewMessageNodes,
	}
	for k, v := range ConvertFocusEvidenceToDiagnostics(focusEvidence) {
		diagnostics[k] = v
	}

	// 添加验证结果诊断
	diagnostics["verified_message_returned"] = strconv.FormatBool(verifiedMessage != nil)
	diagnostics["verification_confidence"] = fmt.Sprintf("%.2f", assessment.Confidence)

	return verifiedMessage, adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
		Confidence: assessment.Confidence,
		ElapsedMs:  elapsedMs,
		Diagnostics: []adapter.Diagnostic{
			{
				Timestamp: time.Now(),
				Level:     "info",
				Message:   "Verification completed with strong validation",
				Context:   diagnostics,
			},
		},
	}
}

// scoreVisualCandidate 计算视觉候选项的评分
func scoreVisualCandidate(rect windows.ConversationRect) int {
	score := 0

	// has_text 优先（最高权重）
	if rect.HasText {
		score += 30
	}
	// has_avatar 次优
	if rect.HasAvatar {
		score += 20
	}
	// has_text + has_avatar 最高（额外加分）
	if rect.HasText && rect.HasAvatar {
		score += 10
	}
	// rect 尺寸合理（宽度 100-300，高度 30-60）
	if rect.Width >= 100 && rect.Width <= 300 && rect.Height >= 30 && rect.Height <= 60 {
		score += 15
	}
	// is_selected 可加分
	if rect.IsSelected {
		score += 10
	}
	// has_unread_dot 可加分
	if rect.HasUnreadDot {
		score += 5
	}
	return score
}

// convertVisionFocusResult 将VisionFocusResult转换为adapter.Result
func (a *WeChatAdapter) convertVisionFocusResult(visionResult windows.VisionFocusResult, startTime time.Time, locateSource string) adapter.Result {
	elapsedMs := time.Since(startTime).Milliseconds()

	// 构建诊断信息
	diagnostics := map[string]string{
		"locate_source":        locateSource,
		"focus_succeeded":      strconv.FormatBool(visionResult.FocusSucceeded),
		"focus_confidence":     fmt.Sprintf("%.2f", visionResult.FocusConfidence),
		"click_strategy":       visionResult.ClickStrategy,
		"click_x":              strconv.Itoa(visionResult.ClickX),
		"click_y":              strconv.Itoa(visionResult.ClickY),
		"click_source":         visionResult.ClickSource,
		"target_index":         strconv.Itoa(visionResult.TargetIndex),
		"success_reasons":      strings.Join(visionResult.SuccessReasons, ", "),
		"processing_time":      visionResult.ProcessingTime.String(),
		"elapsed_ms":           strconv.FormatInt(elapsedMs, 10),
		// Whitelist fields (always present for consistency)
		"evidence_count":       "0",
		"new_message_nodes":    "0",
		"message_content_match": "false",
		"delivery_state":       "unknown",
	}

	// 从VerificationSignals中提取关键信号
	signals := visionResult.VerificationSignals
	// 尝试提取关键信号（使用evaluateFocusSuccess输出的真实key）
	if fullWindowDiff, ok := signals["full_window_diff_percent"].(float64); ok {
		diagnostics["full_window_diff_percent"] = fmt.Sprintf("%.1f", fullWindowDiff)
	}
	if rightSideDiff, ok := signals["right_side_diff_percent"].(float64); ok {
		diagnostics["right_side_diff_percent"] = fmt.Sprintf("%.1f", rightSideDiff)
	}
	if clickedRegionDiff, ok := signals["clicked_region_diff_percent"].(float64); ok {
		diagnostics["clicked_region_diff_percent"] = fmt.Sprintf("%.1f", clickedRegionDiff)
	}
	// 处理差异边界框
	if diffBoundingBox, ok := signals["diff_bounding_box"].([4]int); ok {
		diagnostics["diff_bounding_box_x"] = strconv.Itoa(diffBoundingBox[0])
		diagnostics["diff_bounding_box_y"] = strconv.Itoa(diffBoundingBox[1])
		diagnostics["diff_bounding_box_width"] = strconv.Itoa(diffBoundingBox[2])
		diagnostics["diff_bounding_box_height"] = strconv.Itoa(diffBoundingBox[3])
	}

	var status adapter.Status
	if visionResult.FocusSucceeded {
		status = adapter.StatusSuccess
	} else {
		status = adapter.StatusFailed
	}

	var reasonCode adapter.ReasonCode
	if visionResult.FocusSucceeded {
		reasonCode = adapter.ReasonOK
	} else {
		reasonCode = adapter.ReasonCode("VISION_FOCUS_FAILED")
	}

	return adapter.Result{
		Status:     status,
		ReasonCode: reasonCode,
		Confidence: visionResult.FocusConfidence,
		ElapsedMs:  elapsedMs,
		Diagnostics: []adapter.Diagnostic{
			{
				Timestamp: time.Now(),
				Level:     "info",
				Message:   "Focus completed via vision",
				Context:   diagnostics,
			},
		},
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

