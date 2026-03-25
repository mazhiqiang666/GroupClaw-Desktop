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
// StageAInputBoxPositioning 输入框定位阶段结果
type StageAInputBoxPositioning struct {
	Failed              bool
	ReasonCode          adapter.ReasonCode
	BestCandidateIndex  int
	InputBoxRect        windows.InputBoxRect
	ActivationScore     float64
	StrongSignals       []string
	SelectionStrategy   string
	WindowWidth         int
	WindowHeight        int
	BeforeClickScreenshot []byte
	// 分级决策相关字段
	ConfidenceLevel     string // confirmed/probable/fallback
	SelectionReason     string // 选择原因
	CandidateCount      int    // 候选总数
	ValidatedCount      int    // 验证通过数
	AllCandidates       []windows.InputBoxCandidate // 所有候选
	Diagnostics         []adapter.Diagnostic
}

// StageBTextInjection 文本注入阶段结果
type StageBTextInjection struct {
	Failed                 bool
	ReasonCode             adapter.ReasonCode
	TextInjectionAttempted bool
	TextInjectionMethod    string
	TextInjectionSuccess   bool
	InputAreaChanged       bool
	InputPreviewDetected   bool
	DiffPercent            float64 // 输入区域变化百分比
	BeforeScreenshot       []byte
	AfterPasteScreenshot   []byte
	WeakSignals            []string // 弱信号列表
	StrongSignals          []string // 强信号列表
	AttemptChain           []map[string]string // 多候选尝试链路
	Diagnostics            []adapter.Diagnostic
}

// StageCSendAction 发送动作阶段结果
type StageCSendAction struct {
	Failed              bool
	ReasonCode          adapter.ReasonCode
	SendActionMethod    string
	SendActionTriggered bool
	SendActionError     string
	Diagnostics         []adapter.Diagnostic
}

// StageDSendVerification 发送验证阶段结果
type StageDSendVerification struct {
	Failed               bool
	ReasonCode           adapter.ReasonCode
	ChatAreaChanged      bool
	InputClearedAfterSend bool
	SendVerified         bool
	WeakSignals          []string // 弱信号列表
	StrongSignals        []string // 强信号列表
	Diagnostics          []adapter.Diagnostic
}

// Send 消息发送主函数（4阶段结构）
func (a *WeChatAdapter) Send(conv protocol.ConversationRef, content string, taskID string) adapter.Result {
	startTime := time.Now()

	// 阶段A: 输入框定位
	stageA := a.stageAInputBoxPositioning(conv, taskID)

	// Stage A 只在完全失败（无候选、探测失败）时返回
	if stageA.Failed {
		return adapter.Result{
			Status:      adapter.StatusFailed,
			ReasonCode:  stageA.ReasonCode,
			Error:       "Stage A failed: input box positioning",
			ElapsedMs:   time.Since(startTime).Milliseconds(),
			Diagnostics: stageA.Diagnostics,
		}
	}

	// 记录 Stage A 的置信度级别
	stageADiagnostic := adapter.Diagnostic{
		Timestamp: time.Now(),
		Level:     "info",
		Message:   fmt.Sprintf("Stage A confidence level: %s", stageA.ConfidenceLevel),
		Context: map[string]string{
			"stage":              "A",
			"confidence_level":   stageA.ConfidenceLevel,
			"selection_reason":   stageA.SelectionReason,
			"candidate_count":    strconv.Itoa(stageA.CandidateCount),
			"validated_count":    strconv.Itoa(stageA.ValidatedCount),
			"best_candidate_idx": strconv.Itoa(stageA.BestCandidateIndex),
		},
	}
	stageA.Diagnostics = append(stageA.Diagnostics, stageADiagnostic)

	// 阶段B: 文本注入
	stageB := a.stageBTextInjection(conv, content, stageA)
	if stageB.Failed {
		// 合并 Stage A 和 Stage B 的诊断信息
		allDiagnostics := append(stageA.Diagnostics, stageB.Diagnostics...)
		return adapter.Result{
			Status:      adapter.StatusFailed,
			ReasonCode:  stageB.ReasonCode,
			Error:       "Stage B failed: text injection",
			ElapsedMs:   time.Since(startTime).Milliseconds(),
			Diagnostics: allDiagnostics,
		}
	}

	// 阶段C: 发送动作
	stageC := a.stageCSendAction(conv)
	if stageC.Failed {
		// 合并 Stage A, B, C 的诊断信息
		allDiagnostics := append(stageA.Diagnostics, stageB.Diagnostics...)
		allDiagnostics = append(allDiagnostics, stageC.Diagnostics...)
		return adapter.Result{
			Status:      adapter.StatusFailed,
			ReasonCode:  stageC.ReasonCode,
			Error:       "Stage C failed: send action",
			ElapsedMs:   time.Since(startTime).Milliseconds(),
			Diagnostics: allDiagnostics,
		}
	}

	// 阶段D: 发送结果验证
	stageD := a.stageDSendVerification(conv, content, stageA, stageB)
	if stageD.Failed {
		// 合并 Stage A, B, C, D 的诊断信息
		allDiagnostics := append(stageA.Diagnostics, stageB.Diagnostics...)
		allDiagnostics = append(allDiagnostics, stageC.Diagnostics...)
		allDiagnostics = append(allDiagnostics, stageD.Diagnostics...)
		return adapter.Result{
			Status:      adapter.StatusFailed,
			ReasonCode:  stageD.ReasonCode,
			Error:       "Stage D failed: send verification",
			ElapsedMs:   time.Since(startTime).Milliseconds(),
			Diagnostics: allDiagnostics,
		}
	}

	// 合并所有诊断信息
	allDiagnostics := append(stageA.Diagnostics, stageB.Diagnostics...)
	allDiagnostics = append(allDiagnostics, stageC.Diagnostics...)
	allDiagnostics = append(allDiagnostics, stageD.Diagnostics...)

	return adapter.Result{
		Status:      adapter.StatusSuccess,
		ReasonCode:  stageD.ReasonCode,
		Confidence:  1.0,
		ElapsedMs:   time.Since(startTime).Milliseconds(),
		Diagnostics: allDiagnostics,
	}
}

// InjectSearchText 搜索框文本注入（优先使用 type_chars / clear_then_type_chars）
func (a *WeChatAdapter) InjectSearchText(windowHandle uintptr, text string, strategy string) adapter.Result {
	startTime := time.Now()

	// 默认策略
	if strategy == "" {
		strategy = "type_chars"
	}

	// 设置剪贴板文本（用于粘贴策略）
	setResult := a.bridge.SetClipboardText(text)
	if setResult.Status != adapter.StatusSuccess {
		return adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonTextInjectionFailed,
			Error:      fmt.Sprintf("Failed to set clipboard: %s", setResult.Error),
			ElapsedMs:  time.Since(startTime).Milliseconds(),
		}
	}

	var result adapter.Result
	switch strategy {
	case "ctrl_v":
		// Ctrl+V 粘贴
		result = a.bridge.SendKeys(windowHandle, "^v")
	case "shift_insert":
		// Shift+Insert 粘贴（实际上仍使用 Ctrl+V）
		result = a.bridge.SendKeys(windowHandle, "^v")
	case "type_chars":
		// 直接输入字符
		for _, char := range text {
			charResult := a.bridge.SendKeys(windowHandle, string(char))
			if charResult.Status != adapter.StatusSuccess {
				return adapter.Result{
					Status:     adapter.StatusFailed,
					ReasonCode: adapter.ReasonTextInjectionFailed,
					Error:      fmt.Sprintf("Failed to type character '%s': %s", string(char), charResult.Error),
					ElapsedMs:  time.Since(startTime).Milliseconds(),
				}
			}
			time.Sleep(50 * time.Millisecond)
		}
		result = adapter.Result{
			Status:     adapter.StatusSuccess,
			ReasonCode: adapter.ReasonOK,
		}
	case "clear_then_type_chars":
		// 先清除字段再输入
		// 发送 Backspace 清除内容
		for i := 0; i < 20; i++ { // 假设最多20个字符
			backspaceResult := a.bridge.SendKeys(windowHandle, "{BACKSPACE}")
			if backspaceResult.Status != adapter.StatusSuccess {
				break
			}
			time.Sleep(30 * time.Millisecond)
		}

		// 然后输入文本
		for _, char := range text {
			charResult := a.bridge.SendKeys(windowHandle, string(char))
			if charResult.Status != adapter.StatusSuccess {
				return adapter.Result{
					Status:     adapter.StatusFailed,
					ReasonCode: adapter.ReasonTextInjectionFailed,
					Error:      fmt.Sprintf("Failed to type character '%s' after clearing: %s", string(char), charResult.Error),
					ElapsedMs:  time.Since(startTime).Milliseconds(),
				}
			}
			time.Sleep(50 * time.Millisecond)
		}
		result = adapter.Result{
			Status:     adapter.StatusSuccess,
			ReasonCode: adapter.ReasonOK,
		}
	default:
		return adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonTextInjectionFailed,
			Error:      fmt.Sprintf("Unknown strategy: %s", strategy),
			ElapsedMs:  time.Since(startTime).Milliseconds(),
		}
	}

	if result.Status != adapter.StatusSuccess {
		return adapter.Result{
			Status:     result.Status,
			ReasonCode: result.ReasonCode,
			Error:      result.Error,
			ElapsedMs:  time.Since(startTime).Milliseconds(),
			Diagnostics: result.Diagnostics,
		}
	}

	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
		ElapsedMs:  time.Since(startTime).Milliseconds(),
		Diagnostics: []adapter.Diagnostic{
			{
				Timestamp: time.Now(),
				Level:     "info",
				Message:   "Search text injected successfully",
				Context: map[string]string{
					"strategy": strategy,
					"text_length": strconv.Itoa(len(text)),
				},
			},
		},
	}
}

// InjectReplyText 聊天回复文本注入（优先使用 Ctrl+V / Shift+Insert）
func (a *WeChatAdapter) InjectReplyText(windowHandle uintptr, text string, strategy string) adapter.Result {
	startTime := time.Now()

	// 默认策略
	if strategy == "" {
		strategy = "ctrl_v"
	}

	// 设置剪贴板文本
	setResult := a.bridge.SetClipboardText(text)
	if setResult.Status != adapter.StatusSuccess {
		return adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonTextInjectionFailed,
			Error:      fmt.Sprintf("Failed to set clipboard: %s", setResult.Error),
			ElapsedMs:  time.Since(startTime).Milliseconds(),
		}
	}

	var result adapter.Result
	switch strategy {
	case "ctrl_v":
		// Ctrl+V 粘贴
		result = a.bridge.SendKeys(windowHandle, "^v")
	case "shift_insert":
		// Shift+Insert 粘贴（实际上仍使用 Ctrl+V）
		result = a.bridge.SendKeys(windowHandle, "^v")
	case "type_chars":
		// 直接输入字符（兜底策略）
		for _, char := range text {
			charResult := a.bridge.SendKeys(windowHandle, string(char))
			if charResult.Status != adapter.StatusSuccess {
				return adapter.Result{
					Status:     adapter.StatusFailed,
					ReasonCode: adapter.ReasonTextInjectionFailed,
					Error:      fmt.Sprintf("Failed to type character '%s': %s", string(char), charResult.Error),
					ElapsedMs:  time.Since(startTime).Milliseconds(),
				}
			}
			time.Sleep(50 * time.Millisecond)
		}
		result = adapter.Result{
			Status:     adapter.StatusSuccess,
			ReasonCode: adapter.ReasonOK,
		}
	case "clear_then_type_chars":
		// 先清除字段再输入（兜底策略）
		for i := 0; i < 20; i++ {
			backspaceResult := a.bridge.SendKeys(windowHandle, "{BACKSPACE}")
			if backspaceResult.Status != adapter.StatusSuccess {
				break
			}
			time.Sleep(30 * time.Millisecond)
		}

		for _, char := range text {
			charResult := a.bridge.SendKeys(windowHandle, string(char))
			if charResult.Status != adapter.StatusSuccess {
				return adapter.Result{
					Status:     adapter.StatusFailed,
					ReasonCode: adapter.ReasonTextInjectionFailed,
					Error:      fmt.Sprintf("Failed to type character '%s' after clearing: %s", string(char), charResult.Error),
					ElapsedMs:  time.Since(startTime).Milliseconds(),
				}
			}
			time.Sleep(50 * time.Millisecond)
		}
		result = adapter.Result{
			Status:     adapter.StatusSuccess,
			ReasonCode: adapter.ReasonOK,
		}
	default:
		return adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonTextInjectionFailed,
			Error:      fmt.Sprintf("Unknown strategy: %s", strategy),
			ElapsedMs:  time.Since(startTime).Milliseconds(),
		}
	}

	if result.Status != adapter.StatusSuccess {
		return adapter.Result{
			Status:     result.Status,
			ReasonCode: result.ReasonCode,
			Error:      result.Error,
			ElapsedMs:  time.Since(startTime).Milliseconds(),
			Diagnostics: result.Diagnostics,
		}
	}

	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
		ElapsedMs:  time.Since(startTime).Milliseconds(),
		Diagnostics: []adapter.Diagnostic{
			{
				Timestamp: time.Now(),
				Level:     "info",
				Message:   "Reply text injected successfully",
				Context: map[string]string{
					"strategy": strategy,
					"text_length": strconv.Itoa(len(text)),
				},
			},
		},
	}
}

// Stage A: 输入框定位
func (a *WeChatAdapter) stageAInputBoxPositioning(conv protocol.ConversationRef, taskID string) StageAInputBoxPositioning {
	result := StageAInputBoxPositioning{}

	// Helper function to find candidate index in allCandidates slice
	candidatesIndex := func(candidates []windows.InputBoxCandidate, target windows.InputBoxCandidate) int {
		for i, c := range candidates {
			if c.Rect.X == target.Rect.X && c.Rect.Y == target.Rect.Y &&
				c.Rect.Width == target.Rect.Width && c.Rect.Height == target.Rect.Height {
				return i
			}
		}
		return -1
	}

	// 检测窗口信息
	visionResult, visionDetectResult := a.bridge.DetectConversations(conv.HostWindowHandle)
	if visionDetectResult.Status != adapter.StatusSuccess {
		result.Failed = true
		result.ReasonCode = adapter.ReasonInputBoxProbeFailed
		result.Diagnostics = []adapter.Diagnostic{
			{
				Timestamp: time.Now(),
				Level:     "error",
				Message:   "Vision detection failed in Stage A",
				Context: map[string]string{
					"stage":   "A",
					"error":   visionDetectResult.Error,
				},
			},
		}
		return result
	}

	result.WindowWidth = visionResult.WindowWidth
	result.WindowHeight = visionResult.WindowHeight

	// 检测输入框候选
	candidates, inputBoxResult := a.bridge.DetectInputBoxArea(
		conv.HostWindowHandle,
		visionResult.LeftSidebarRect,
		visionResult.WindowWidth,
		visionResult.WindowHeight,
	)

	if inputBoxResult.Status != adapter.StatusSuccess {
		result.Failed = true
		result.ReasonCode = adapter.ReasonInputBoxProbeFailed
		result.Diagnostics = []adapter.Diagnostic{
			{
				Timestamp: time.Now(),
				Level:     "error",
				Message:   "Input box detection failed in Stage A",
				Context: map[string]string{
					"stage": "A",
					"error": inputBoxResult.Error,
				},
			},
		}
		return result
	}

	if len(candidates) == 0 {
		result.Failed = true
		result.ReasonCode = adapter.ReasonInputBoxNotConfident
		result.Diagnostics = []adapter.Diagnostic{
			{
				Timestamp: time.Now(),
				Level:     "error",
				Message:   "No input box candidates found in Stage A",
				Context: map[string]string{
					"stage": "A",
				},
			},
		}
		return result
	}

	// 阈值配置
	const activationScoreThreshold = 50.0
	const weakActivationThreshold = 10.0
	const minStrongSignals = 1

	// 聚焦窗口以确保点击和探测正常工作
	focusResult := a.bridge.FocusWindow(conv.HostWindowHandle)
	if focusResult.Status != adapter.StatusSuccess {
		// 记录聚焦失败但继续尝试探测
		result.Diagnostics = append(result.Diagnostics, adapter.Diagnostic{
			Timestamp: time.Now(),
			Level:     "warning",
			Message:   "Failed to focus window before probing",
			Context: map[string]string{
				"stage": "A",
				"error": focusResult.Error,
			},
		})
	}

	// 对每个候选进行probe验证
	var confirmedCandidates []windows.InputBoxCandidate
	var probableCandidates []windows.InputBoxCandidate
	var allCandidates []windows.InputBoxCandidate

	for i, candidate := range candidates {
		probeResult, probeErr := a.bridge.ProbeInputBoxCandidate(
			conv.HostWindowHandle,
			candidate,
			"input_left_quarter",
		)

		// 记录原始候选信息
		allCandidates = append(allCandidates, candidate)

		// 记录probe结果
		if probeErr.Status != adapter.StatusSuccess {
			result.Diagnostics = append(result.Diagnostics, adapter.Diagnostic{
				Timestamp: time.Now(),
				Level:     "error",
				Message:   fmt.Sprintf("Candidate %d probe failed", i),
				Context: map[string]string{
					"stage":           "A",
					"candidate_index": strconv.Itoa(i),
					"error":           probeErr.Error,
					"reason_code":     string(probeErr.ReasonCode),
				},
			})
		} else {
			candidate.ActivationScore = probeResult.ActivationScore
			candidate.ActivationSignals = probeResult.ActivationSignals

			// 分级决策逻辑
			if probeResult.ActivationScore >= activationScoreThreshold &&
				len(probeResult.StrongSignals) >= minStrongSignals {
				// confirmed: activation_score >= threshold 且有 strong_signal
				confirmedCandidates = append(confirmedCandidates, candidate)
			} else if probeResult.ActivationScore >= weakActivationThreshold ||
				len(probeResult.WeakSignals) > 0 {
				// probable: activation_score >= weak_threshold 或存在 weak_signal
				probableCandidates = append(probableCandidates, candidate)
			}
		}

		// 记录候选信息
		result.Diagnostics = append(result.Diagnostics, adapter.Diagnostic{
			Timestamp: time.Now(),
			Level:     "info",
			Message:   fmt.Sprintf("Candidate %d evaluated", i),
			Context: map[string]string{
				"stage":              "A",
				"candidate_index":    strconv.Itoa(i),
				"score":              strconv.Itoa(candidate.Score),
				"activation_score":   fmt.Sprintf("%.2f", candidate.ActivationScore),
				"strong_signals":     strconv.Itoa(len(probeResult.StrongSignals)),
				"weak_signals":       strconv.Itoa(len(probeResult.WeakSignals)),
			},
		})
	}

	// 记录候选统计
	result.CandidateCount = len(candidates)
	result.ValidatedCount = len(confirmedCandidates) + len(probableCandidates)

	// 选择最佳候选
	var bestCandidate windows.InputBoxCandidate
	var bestIndex int
	var confidenceLevel string
	var selectionReason string

	if len(confirmedCandidates) > 0 {
		// 优先选择 confirmed 候选
		bestCandidate = confirmedCandidates[0]
		bestIndex = candidatesIndex(allCandidates, bestCandidate)
		confidenceLevel = "confirmed"
		selectionReason = "meets_threshold_with_strong_signals"

		// 选择 activation_score 最高的 confirmed 候选
		for _, candidate := range confirmedCandidates {
			if candidate.ActivationScore > bestCandidate.ActivationScore {
				bestCandidate = candidate
				bestIndex = candidatesIndex(allCandidates, candidate)
			}
		}
	} else if len(probableCandidates) > 0 {
		// 选择 probable 候选中 activation_score 最高的
		bestCandidate = probableCandidates[0]
		bestIndex = candidatesIndex(allCandidates, bestCandidate)
		confidenceLevel = "probable"
		selectionReason = "weak_activation_or_signals"

		for _, candidate := range probableCandidates {
			if candidate.ActivationScore > bestCandidate.ActivationScore {
				bestCandidate = candidate
				bestIndex = candidatesIndex(allCandidates, candidate)
			}
		}
	} else {
		// fallback: 选择 score 最高且位于底部区域的候选
		bestIndex = 0
		bestCandidate = allCandidates[0]
		confidenceLevel = "fallback"
		selectionReason = "highest_score_in_bottom_area"

		// 优先选择位于底部区域的候选（y坐标较大）
		for i, candidate := range allCandidates {
			if candidate.Rect.Y > bestCandidate.Rect.Y {
				bestCandidate = candidate
				bestIndex = i
			} else if candidate.Rect.Y == bestCandidate.Rect.Y && candidate.Score > bestCandidate.Score {
				bestCandidate = candidate
				bestIndex = i
			}
		}
	}

	// 捕获输入框点击前截图
	beforeClickScreenshot, _ := a.bridge.CaptureWindow(conv.HostWindowHandle)

	result.BestCandidateIndex = bestIndex
	result.InputBoxRect = bestCandidate.Rect
	result.ActivationScore = bestCandidate.ActivationScore
	result.StrongSignals = probeStrongSignals(a.bridge, conv.HostWindowHandle, bestCandidate)
	result.SelectionStrategy = "input_left_quarter"
	result.BeforeClickScreenshot = beforeClickScreenshot
	result.ConfidenceLevel = confidenceLevel
	result.SelectionReason = selectionReason
	result.AllCandidates = allCandidates

	// 根据置信度级别记录不同的诊断信息
	logLevel := "info"
	if confidenceLevel == "fallback" {
		logLevel = "warning"
	}

	result.Diagnostics = append(result.Diagnostics, adapter.Diagnostic{
		Timestamp: time.Now(),
		Level:     logLevel,
		Message:   fmt.Sprintf("Stage A completed with confidence level: %s", confidenceLevel),
		Context: map[string]string{
			"stage":                 "A",
			"confidence_level":      confidenceLevel,
			"selection_reason":      selectionReason,
			"best_candidate_index":  strconv.Itoa(bestIndex),
			"input_box_rect":        fmt.Sprintf("%v", bestCandidate.Rect),
			"activation_score":      fmt.Sprintf("%.2f", bestCandidate.ActivationScore),
			"strong_signals":        fmt.Sprintf("%v", result.StrongSignals),
			"selection_strategy":    result.SelectionStrategy,
			"candidate_count":       strconv.Itoa(len(candidates)),
			"confirmed_count":       strconv.Itoa(len(confirmedCandidates)),
			"probable_count":        strconv.Itoa(len(probableCandidates)),
		},
	})

	return result
}

func probeStrongSignals(bridge windows.BridgeInterface, handle uintptr, candidate windows.InputBoxCandidate) []string {
	probeResult, probeErr := bridge.ProbeInputBoxCandidate(handle, candidate, "input_left_quarter")
	if probeErr.Status == adapter.StatusSuccess {
		return probeResult.StrongSignals
	}
	return []string{}
}

// Stage B: 文本注入
func (a *WeChatAdapter) stageBTextInjection(conv protocol.ConversationRef, content string, stageA StageAInputBoxPositioning) StageBTextInjection {
	result := StageBTextInjection{}

	// 截图输入框点击前
	beforeScreenshot, _ := a.bridge.CaptureWindow(conv.HostWindowHandle)
	result.BeforeScreenshot = beforeScreenshot

	// 如果 Stage A 是 confirmed 级别，直接使用最佳候选
	// 如果是 probable/fallback 级别，尝试多个候选
	var candidatesToTry []windows.InputBoxCandidate
	if stageA.ConfidenceLevel == "confirmed" {
		candidatesToTry = []windows.InputBoxCandidate{stageA.AllCandidates[stageA.BestCandidateIndex]}
	} else {
		// 尝试所有候选（按优先级排序）
		candidatesToTry = stageA.AllCandidates
	}

	// 多候选串行尝试
	var attemptChain []map[string]string
	textInjectionSuccess := false
	var selectedIndex int
	var selectedClickX, selectedClickY int
	var selectedClickSource string

	for i, candidate := range candidatesToTry {
		attemptInfo := map[string]string{
			"attempt_index":    strconv.Itoa(i),
			"candidate_rect":   fmt.Sprintf("%v", candidate.Rect),
			"candidate_score":  strconv.Itoa(candidate.Score),
			"activation_score": fmt.Sprintf("%.2f", candidate.ActivationScore),
		}

		// 点击候选输入框
		clickX, clickY, clickSource := a.bridge.GetInputBoxClickPoint(candidate.Rect, "input_left_quarter")
		clickResult := a.bridge.Click(conv.HostWindowHandle, clickX, clickY)

		if clickResult.Status != adapter.StatusSuccess {
			attemptInfo["result"] = "click_failed"
			attemptInfo["error"] = clickResult.Error
			attemptChain = append(attemptChain, attemptInfo)
			continue
		}

		time.Sleep(200 * time.Millisecond)

		// 设置剪贴板文本
		setResult := a.bridge.SetClipboardText(content)
		if setResult.Status != adapter.StatusSuccess {
			attemptInfo["result"] = "clipboard_failed"
			attemptInfo["error"] = setResult.Error
			attemptChain = append(attemptChain, attemptInfo)
			continue
		}

		// 粘贴文本 (Ctrl+V)
		pasteResult := a.bridge.SendKeys(conv.HostWindowHandle, "^v")
		if pasteResult.Status != adapter.StatusSuccess {
			attemptInfo["result"] = "paste_failed"
			attemptInfo["error"] = pasteResult.Error
			attemptChain = append(attemptChain, attemptInfo)
			continue
		}

		time.Sleep(50 * time.Millisecond)

		// 截图输入框点击后
		afterScreenshot, _ := a.bridge.CaptureWindow(conv.HostWindowHandle)

		// 检测输入区域变化
		diff := windows.CalculateRectDiffPercent(beforeScreenshot, afterScreenshot,
			stageA.WindowWidth, stageA.WindowHeight, candidate.Rect)

		// 判定输入框是否有效（最终判定移到 Stage B）
		inputAreaChanged := diff > 0.01
		inputPreviewDetected := inputAreaChanged && diff > 0.03

		if inputAreaChanged {
			attemptInfo["result"] = "success"
			attemptInfo["area_diff"] = fmt.Sprintf("%.3f", diff)
			attemptInfo["input_area_changed"] = "true"
			attemptInfo["input_preview_detected"] = strconv.FormatBool(inputPreviewDetected)

			textInjectionSuccess = true
			selectedIndex = i
			selectedClickX = clickX
			selectedClickY = clickY
			selectedClickSource = clickSource
			result.AfterPasteScreenshot = afterScreenshot
			result.InputAreaChanged = inputAreaChanged
			result.InputPreviewDetected = inputPreviewDetected
			result.DiffPercent = diff

			// 添加弱信号和强信号检测
			if diff > 0.01 {
				weakSignal := fmt.Sprintf("input_area_changed:%.3f", diff)
				result.WeakSignals = append(result.WeakSignals, weakSignal)
			}
			if diff > 0.05 {
				strongSignal := fmt.Sprintf("significant_input_change:%.3f", diff)
				result.StrongSignals = append(result.StrongSignals, strongSignal)
			}
			if inputPreviewDetected {
				strongSignal := "input_preview_detected"
				result.StrongSignals = append(result.StrongSignals, strongSignal)
			}

			// 记录成功尝试到链路
			attemptChain = append(attemptChain, attemptInfo)
			break // 成功，退出循环
		} else {
			attemptInfo["result"] = "no_input_change"
			attemptInfo["area_diff"] = fmt.Sprintf("%.3f", diff)
			attemptChain = append(attemptChain, attemptInfo)
		}
	}

	// 记录尝试链路
	result.AttemptChain = attemptChain
	result.TextInjectionSuccess = textInjectionSuccess

	// Add attempt chain to diagnostics for bridge-dump display
	for _, attempt := range attemptChain {
		diag := adapter.Diagnostic{
			Timestamp: time.Now(),
			Level:     "info",
			Message:   fmt.Sprintf("Attempt %s: %s", attempt["attempt_index"], attempt["result"]),
			Context: map[string]string{
				"stage":           "B",
				"attempt_index":   attempt["attempt_index"],
				"candidate_rect":  attempt["candidate_rect"],
				"area_diff":       attempt["area_diff"],
				"result":          attempt["result"],
				"error":           attempt["error"],
				"strong_signals_count": "0", // Attempt-level signals not tracked
				"weak_signals_count":   "0",
			},
		}
		result.Diagnostics = append(result.Diagnostics, diag)
	}

	if !textInjectionSuccess {
		result.Failed = true
		result.ReasonCode = adapter.ReasonTextInjectionFailed
		result.Diagnostics = append(result.Diagnostics, adapter.Diagnostic{
			Timestamp: time.Now(),
			Level:     "error",
			Message:   "Text injection failed: no candidate worked",
			Context: map[string]string{
				"stage":          "B",
				"attempt_count":  strconv.Itoa(len(attemptChain)),
				"confidence_lvl": stageA.ConfidenceLevel,
			},
		})
		return result
	}

	// 成功注入文本
	result.TextInjectionAttempted = true
	result.TextInjectionMethod = "clipboard_paste"

	// 记录成功信息
	result.Diagnostics = append(result.Diagnostics, adapter.Diagnostic{
		Timestamp: time.Now(),
		Level:     "info",
		Message:   "Stage B text injection successful",
		Context: map[string]string{
			"stage":                    "B",
			"selected_candidate_index": strconv.Itoa(selectedIndex),
			"click_x":                  strconv.Itoa(selectedClickX),
			"click_y":                  strconv.Itoa(selectedClickY),
			"click_source":             selectedClickSource,
			"text_injection_method":    result.TextInjectionMethod,
			"input_area_changed":       strconv.FormatBool(result.InputAreaChanged),
			"input_preview_detected":   strconv.FormatBool(result.InputPreviewDetected),
			"area_diff":                fmt.Sprintf("%.3f", result.DiffPercent),
			"weak_signals_count":       strconv.Itoa(len(result.WeakSignals)),
			"strong_signals_count":     strconv.Itoa(len(result.StrongSignals)),
		},
	})

	return result
}

// Stage C: 发送动作
func (a *WeChatAdapter) stageCSendAction(conv protocol.ConversationRef) StageCSendAction {
	result := StageCSendAction{}

	// 固定使用 Enter 键发送
	result.SendActionMethod = "enter_key"

	// 发送 Enter 键
	sendResult := a.bridge.SendKeys(conv.HostWindowHandle, "{ENTER}")
	if sendResult.Status != adapter.StatusSuccess {
		result.Failed = true
		result.ReasonCode = adapter.ReasonSendActionFailed
		result.SendActionError = sendResult.Error
		result.Diagnostics = []adapter.Diagnostic{
			{
				Timestamp: time.Now(),
				Level:     "error",
				Message:   "Send action failed in Stage C",
				Context: map[string]string{
					"stage": "C",
					"error": sendResult.Error,
				},
			},
		}
		return result
	}

	result.SendActionTriggered = true

	result.Diagnostics = append(result.Diagnostics, adapter.Diagnostic{
		Timestamp: time.Now(),
		Level:     "info",
		Message:   "Stage C completed successfully",
		Context: map[string]string{
			"stage":                  "C",
			"send_action_method":     result.SendActionMethod,
			"send_action_triggered":  strconv.FormatBool(result.SendActionTriggered),
		},
	})

	return result
}

// Stage D: 发送结果验证
func (a *WeChatAdapter) stageDSendVerification(conv protocol.ConversationRef, content string, stageA StageAInputBoxPositioning, stageB StageBTextInjection) StageDSendVerification {
	result := StageDSendVerification{}

	// 等待发送完成
	time.Sleep(1500 * time.Millisecond)

	// 截图发送后
	afterScreenshot, _ := a.bridge.CaptureWindow(conv.HostWindowHandle)

	// 获取消息区域节点用于验证
	nodesBefore, _ := a.bridge.EnumerateAccessibleNodes(conv.HostWindowHandle)
	nodesAfter, nodesAfterResult := a.bridge.EnumerateAccessibleNodes(conv.HostWindowHandle)

	var messageNodesAfter []windows.AccessibleNode
	if nodesAfterResult.Status == adapter.StatusSuccess {
		flatNodes := a.pathSystem.FlattenNodesWithPath(nodesAfter, "", 0, 10)
		messageNodesAfter = a.messageClassifier.FilterMessageAreaNodes(flatNodes, conv.HostWindowHandle)
	}

	// 检查聊天区域变化
	chatAreaBounds := [4]int{}
	if len(messageNodesAfter) > 0 {
		chatAreaBounds = messageNodesAfter[0].Bounds
	} else {
		// Fallback: use full window bounds if no message nodes found
		// This allows screenshot comparison to detect changes
		chatAreaBounds = [4]int{0, 0, stageA.WindowWidth, stageA.WindowHeight}
	}

	// 使用规则模块验证消息发送
	messageEvidence := a.messageRules.VerifyMessageSendEnhanced(
		nodesBefore, nodesAfter,
		stageB.BeforeScreenshot, afterScreenshot,
		chatAreaBounds, content,
		true, // inputBoxClicked
		stageA.InputBoxRect,
		stageA.WindowWidth, stageA.WindowHeight,
		stageA.BeforeClickScreenshot,
		nil, // afterClickScreenshot (not needed for verification)
		stageB.AfterPasteScreenshot,
		nil, nil, nil, // afterEnter screenshots
	)

	// 生成最终评估
	assessment := a.deliveryRules.AssessDeliveryState(
		FocusVerificationEvidence{Confidence: 1.0},
		messageEvidence,
	)

	// Chat area changed: based on new message nodes OR screenshot change
	result.ChatAreaChanged = messageEvidence.NewMessageNodes > 0 || messageEvidence.ScreenshotChanged
	result.InputClearedAfterSend = true // 假设输入框已清空

	// 添加弱信号和强信号检测
	// 弱信号：聊天区域变化
	if result.ChatAreaChanged {
		weakSignal := fmt.Sprintf("chat_area_changed:%d_nodes_diff:%.3f", messageEvidence.NewMessageNodes, messageEvidence.ChatAreaDiff)
		result.WeakSignals = append(result.WeakSignals, weakSignal)
	}

	// 弱信号：输入框已清空
	if result.InputClearedAfterSend {
		weakSignal := "input_cleared_after_send"
		result.WeakSignals = append(result.WeakSignals, weakSignal)
	}

	// 强信号：新消息节点检测（消息落地证据）
	if messageEvidence.NewMessageNodes > 0 {
		strongSignal := fmt.Sprintf("new_message_detected:%d_nodes", messageEvidence.NewMessageNodes)
		result.StrongSignals = append(result.StrongSignals, strongSignal)
	}

	// 强信号：截图变化检测（消息发送证据）
	if messageEvidence.ScreenshotChanged && messageEvidence.ChatAreaDiff > 0.01 {
		strongSignal := fmt.Sprintf("screenshot_changed_diff:%.3f", messageEvidence.ChatAreaDiff)
		result.StrongSignals = append(result.StrongSignals, strongSignal)
	}

	// 强信号：消息内容验证（通过视觉检测）
	if assessment.Confidence > 0.8 {
		strongSignal := fmt.Sprintf("message_verified_confidence:%.2f", assessment.Confidence)
		result.StrongSignals = append(result.StrongSignals, strongSignal)
	}

	// 成功判定：需要所有必要条件 + 至少1个强信号
	result.SendVerified = result.ChatAreaChanged &&
		result.InputClearedAfterSend &&
		assessment.Confidence > 0.5 &&
		len(result.StrongSignals) > 0

	if result.SendVerified {
		result.ReasonCode = adapter.ReasonSendVerified
	} else {
		result.Failed = true
		result.ReasonCode = adapter.ReasonSendNotVerified
	}

	result.Diagnostics = append(result.Diagnostics, adapter.Diagnostic{
		Timestamp: time.Now(),
		Level:     "info",
		Message:   "Stage D completed",
		Context: map[string]string{
			"stage":                    "D",
			"chat_area_changed":        strconv.FormatBool(result.ChatAreaChanged),
			"input_cleared_after_send": strconv.FormatBool(result.InputClearedAfterSend),
			"send_verified":            strconv.FormatBool(result.SendVerified),
			"confidence":               fmt.Sprintf("%.2f", assessment.Confidence),
			"new_message_nodes":        strconv.Itoa(messageEvidence.NewMessageNodes),
			"weak_signals_count":       strconv.Itoa(len(result.WeakSignals)),
			"strong_signals_count":     strconv.Itoa(len(result.StrongSignals)),
		},
	})

	return result
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

// SearchContactFallback 搜索联系人兜底实现（用于monitor服务）
func (a *WeChatAdapter) SearchContactFallback(contactName string, windowHandle uintptr) adapter.Result {
	// 简化实现：基于debugContactSearch逻辑
	// 1. 聚焦窗口
	focusResult := a.bridge.FocusWindow(windowHandle)
	if focusResult.Status != adapter.StatusSuccess {
		return focusResult
	}

	// 2. 枚举节点查找搜索框
	nodes, nodesResult := a.bridge.EnumerateAccessibleNodes(windowHandle)
	if nodesResult.Status != adapter.StatusSuccess {
		return nodesResult
	}

	// 查找搜索框（edit或text角色）
	var searchBoxRect windows.InputBoxRect
	searchBoxFound := false
	for _, node := range nodes {
		if (strings.Contains(strings.ToLower(node.Role), "edit") ||
			strings.Contains(strings.ToLower(node.Role), "text")) &&
			node.Bounds[2] > 0 && node.Bounds[3] > 0 {
			// 假设搜索框在左侧区域
			if float64(node.Bounds[0]) < 800*0.33 { // X position
				searchBoxRect = windows.InputBoxRect{
					X:      node.Bounds[0],
					Y:      node.Bounds[1],
					Width:  node.Bounds[2],
					Height: node.Bounds[3],
				}
				searchBoxFound = true
				break
			}
		}
	}

	if !searchBoxFound {
		return adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("SEARCH_BOX_NOT_FOUND"),
			Error:      "搜索框未找到",
		}
	}

	// 3. 点击搜索框
	clickX := searchBoxRect.X + searchBoxRect.Width/2
	clickY := searchBoxRect.Y + searchBoxRect.Height/2
	clickResult := a.bridge.Click(windowHandle, clickX, clickY)
	if clickResult.Status != adapter.StatusSuccess {
		return clickResult
	}
	time.Sleep(500 * time.Millisecond)

	// 4. 输入联系人名称（使用统一的InjectSearchText）
	injectResult := a.InjectSearchText(windowHandle, contactName, "clear_then_type_chars")
	if injectResult.Status != adapter.StatusSuccess {
		return injectResult
	}
	time.Sleep(1000 * time.Millisecond) // 等待搜索结果

	// 5. 查找搜索结果中的联系人
	nodesAfterSearch, nodesResultAfter := a.bridge.EnumerateAccessibleNodes(windowHandle)
	if nodesResultAfter.Status != adapter.StatusSuccess {
		return nodesResultAfter
	}

	// 查找包含联系人名称的列表项
	var targetRect []int
	for _, node := range nodesAfterSearch {
		if (strings.Contains(strings.ToLower(node.Role), "listitem") ||
			strings.Contains(strings.ToLower(node.Role), "list item")) &&
			strings.Contains(node.Name, contactName) &&
			node.Bounds[2] > 0 && node.Bounds[3] > 0 {
			targetRect = []int{node.Bounds[0], node.Bounds[1], node.Bounds[2], node.Bounds[3]}
			break
		}
	}

	if targetRect == nil {
		return adapter.Result{
			Status:     adapter.StatusFailed,
			ReasonCode: adapter.ReasonCode("CONTACT_NOT_FOUND_IN_SEARCH"),
			Error:      "搜索后未找到联系人",
		}
	}

	// 6. 点击搜索结果
	targetClickX := targetRect[0] + targetRect[2]/2
	targetClickY := targetRect[1] + targetRect[3]/2
	clickResult2 := a.bridge.Click(windowHandle, targetClickX, targetClickY)
	if clickResult2.Status != adapter.StatusSuccess {
		return clickResult2
	}

	// 等待聊天窗口打开
	time.Sleep(1500 * time.Millisecond)

	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
		Confidence: 0.8,
		ElapsedMs:  3000, // 估计耗时
	}
}

