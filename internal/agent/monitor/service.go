package monitor

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mazhiqiang666/GroupClaw-Desktop/internal/agent/adapter"
	"github.com/mazhiqiang666/GroupClaw-Desktop/internal/agent/session"
	"github.com/mazhiqiang666/GroupClaw-Desktop/pkg/protocol"
)

// MonitorService 监控服务
type MonitorService struct {
	adapter       adapter.ChatAdapter
	sessionMgr    *session.SessionManager
	agentClient   RemoteAgentClient
	pollInterval  time.Duration
	windowHandle  uintptr
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	mu            sync.RWMutex
	isRunning     bool
	maxRetries    int
	operationTimeout time.Duration
	dryRun        bool
	pollRound     int64 // 监控轮次计数器
}

// RemoteAgentClient 远端agent客户端接口
type RemoteAgentClient interface {
	GetReply(ctx context.Context, context AgentContext) (string, error)
	GetEndpoint() string
}

// AgentContext 发送给远端agent的上下文
type AgentContext struct {
	ContactID     string             `json:"contact_id"`
	ContactName   string             `json:"contact_name"`
	MessageHistory []session.Message `json:"message_history"`
	UnreadCount   int                `json:"unread_count"`
	LastReplyTime time.Time          `json:"last_reply_time,omitempty"`
	Timestamp     time.Time          `json:"timestamp"`
}

// Config 监控服务配置
type Config struct {
	PollInterval      time.Duration `yaml:"poll_interval"`
	MaxRetries        int           `yaml:"max_retries"`
	OperationTimeout  time.Duration `yaml:"operation_timeout"`
	AgentEndpoint     string        `yaml:"agent_endpoint"`
	DryRun           bool           `yaml:"dry_run"`
}

// DefaultConfig 默认配置
var DefaultConfig = Config{
	PollInterval:     5 * time.Second,
	MaxRetries:       3,
	OperationTimeout: 10 * time.Second,
	AgentEndpoint:    "http://localhost:8080/api/reply",
	DryRun:           false,
}

// NewMonitorService 创建新的监控服务
func NewMonitorService(
	adapter adapter.ChatAdapter,
	sessionMgr *session.SessionManager,
	agentClient RemoteAgentClient,
	config Config,
) *MonitorService {
	ctx, cancel := context.WithCancel(context.Background())

	return &MonitorService{
		adapter:         adapter,
		sessionMgr:      sessionMgr,
		agentClient:     agentClient,
		pollInterval:    config.PollInterval,
		ctx:             ctx,
		cancel:          cancel,
		isRunning:       false,
		maxRetries:      config.MaxRetries,
		operationTimeout: config.OperationTimeout,
		dryRun:          config.DryRun,
		pollRound:       0,
	}
}

// Start 启动监控服务
func (ms *MonitorService) Start() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if ms.isRunning {
		return fmt.Errorf("monitor service is already running")
	}

	// 检测应用实例
	instances, result := ms.adapter.Detect()
	if result.Status != adapter.StatusSuccess || len(instances) == 0 {
		return fmt.Errorf("failed to detect application instances: %s", result.Error)
	}

	// windowHandle will be obtained from conversation ref when needed
	ms.windowHandle = 0
	log.Printf("监控服务启动，检测到 %d 个应用实例", len(instances))

	ms.isRunning = true
	ms.wg.Add(1)

	go ms.monitorLoop()

	log.Println("监控服务启动成功")
	return nil
}

// Stop 停止监控服务
func (ms *MonitorService) Stop() {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if !ms.isRunning {
		return
	}

	log.Println("正在停止监控服务...")
	ms.cancel()
	ms.isRunning = false

	// 等待监控循环结束
	done := make(chan struct{})
	go func() {
		ms.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("监控服务已停止")
	case <-time.After(5 * time.Second):
		log.Println("监控服务停止超时")
	}
}

// IsRunning 检查是否正在运行
func (ms *MonitorService) IsRunning() bool {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return ms.isRunning
}

// monitorLoop 监控主循环
func (ms *MonitorService) monitorLoop() {
	defer ms.wg.Done()

	log.Println("监控循环开始")

	ticker := time.NewTicker(ms.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ms.ctx.Done():
			log.Println("监控循环接收到停止信号")
			return
		case <-ticker.C:
			ms.monitorCycle()
		}
	}
}

// monitorCycle 单次监控周期（实现8个步骤）
func (ms *MonitorService) monitorCycle() {
	startTime := time.Now()
	round := atomic.AddInt64(&ms.pollRound, 1)
	log.Printf("[MONITOR] poll_round=%d, started", round)

	defer func() {
		if r := recover(); r != nil {
			log.Printf("[MONITOR] poll_round=%d, error=panic, reason=%v", round, r)
		}
		elapsed := time.Since(startTime)
		log.Printf("[MONITOR] poll_round=%d, completed, elapsed=%v", round, elapsed)
	}()

	// 步骤1: 监测联系人列表
	contacts, err := ms.monitorContactList()
	if err != nil {
		log.Printf("[MONITOR] poll_round=%d, stage=contact_detection, error=%v", round, err)
		return
	}
	log.Printf("[MONITOR] poll_round=%d, stage=contact_detection, detected_contacts=%d", round, len(contacts))

	// 查找有未读消息的联系人
	unreadContacts := ms.filterUnreadContacts(contacts)
	if len(unreadContacts) == 0 {
		log.Printf("[MONITOR] poll_round=%d, stage=unread_filter, reason=no_unread_contacts", round)
		return
	}
	log.Printf("[MONITOR] poll_round=%d, stage=unread_filter, unread_contacts=%d", round, len(unreadContacts))

	// 处理每个有未读消息的联系人（按未读数量排序）
	for i, contact := range unreadContacts {
		contactRound := fmt.Sprintf("%d.%d", round, i+1)
		log.Printf("[MONITOR] poll_round=%s, stage=contact_processing, selected_contact=%s, unread_count=%d",
			contactRound, contact.Name, contact.UnreadCount)

		if err := ms.processContact(contact, contactRound); err != nil {
			log.Printf("[MONITOR] poll_round=%s, stage=contact_processing, error=%v", contactRound, err)
			// 继续处理下一个联系人
			continue
		}
	}
}

// monitorContactList 监测联系人列表（步骤1）
func (ms *MonitorService) monitorContactList() ([]ContactInfo, error) {
	log.Println("步骤1: 监测联系人列表")

	// 使用适配器扫描会话
	instances, detectResult := ms.adapter.Detect()
	if detectResult.Status != adapter.StatusSuccess || len(instances) == 0 {
		return nil, fmt.Errorf("检测应用实例失败: %s", detectResult.Error)
	}

	conversations, scanResult := ms.adapter.Scan(instances[0])
	if scanResult.Status != adapter.StatusSuccess {
		return nil, fmt.Errorf("扫描会话失败: %s", scanResult.Error)
	}

	var contacts []ContactInfo
	for _, conv := range conversations {
		contacts = append(contacts, ContactInfo{
			ID:           conv.DisplayName, // 使用DisplayName作为ID
			Name:         conv.DisplayName,
			UnreadCount:  estimateUnreadCount(conv), // 需要实现估计未读数
			Conversation: conv,
		})
	}

	log.Printf("检测到 %d 个联系人", len(contacts))
	return contacts, nil
}

// filterUnreadContacts 过滤有未读消息的联系人
func (ms *MonitorService) filterUnreadContacts(contacts []ContactInfo) []ContactInfo {
	var unread []ContactInfo
	for _, contact := range contacts {
		if contact.UnreadCount > 0 {
			unread = append(unread, contact)
		}
	}
	return unread
}

// processContact 处理单个联系人（步骤2-8）
func (ms *MonitorService) processContact(contact ContactInfo, pollRound string) error {
	log.Printf("[MONITOR] poll_round=%s, stage=contact_selected, selected_contact=%s, unread_count=%d",
		pollRound, contact.Name, contact.UnreadCount)

	// 步骤2: 打开有新消息的联系人
	convRef, err := ms.openContact(contact)
	if err != nil {
		log.Printf("[MONITOR] poll_round=%s, stage=chat_open, reason=open_failed, error=%v", pollRound, err)
		return fmt.Errorf("打开联系人失败: %v", err)
	}
	log.Printf("[MONITOR] poll_round=%s, stage=chat_open, chat_open_verified=true", pollRound)

	// 更新会话引用
	if err := ms.sessionMgr.SetConversationRef(contact.ID, &convRef); err != nil {
		log.Printf("[MONITOR] poll_round=%s, stage=session_update, reason=ref_update_failed, error=%v", pollRound, err)
	}

	// 步骤3: 读取新增消息
	messages, err := ms.readNewMessages(convRef, contact.ID)
	if err != nil {
		log.Printf("[MONITOR] poll_round=%s, stage=message_read, reason=read_failed, error=%v", pollRound, err)
		return fmt.Errorf("读取消息失败: %v", err)
	}

	if len(messages) == 0 {
		log.Printf("[MONITOR] poll_round=%s, stage=message_read, reason=no_new_messages", pollRound)
		return nil
	}

	log.Printf("[MONITOR] poll_round=%s, stage=message_read, new_messages_count=%d", pollRound, len(messages))

	// 步骤4: 更新该联系人的session
	if err := ms.updateSessionWithMessages(contact.ID, messages); err != nil {
		log.Printf("[MONITOR] poll_round=%s, stage=session_update, reason=message_update_failed, error=%v", pollRound, err)
		return fmt.Errorf("更新会话失败: %v", err)
	}
	log.Printf("[MONITOR] poll_round=%s, stage=session_update, session_updated=true", pollRound)

	// 步骤5: 调用远端agent获取回复
	replyContent, err := ms.callRemoteAgent(contact.ID)
	if err != nil {
		log.Printf("[MONITOR] poll_round=%s, stage=agent_request, reason=agent_failed, error=%v", pollRound, err)
		return fmt.Errorf("获取回复失败: %v", err)
	}

	if replyContent == "" {
		log.Printf("[MONITOR] poll_round=%s, stage=agent_request, reason=empty_reply", pollRound)
		return nil
	}
	log.Printf("[MONITOR] poll_round=%s, stage=agent_request, agent_reply_received=true, reply_length=%d", pollRound, len(replyContent))

	// 步骤6: 确认当前聊天框仍属于该联系人
	if err := ms.verifyCurrentChat(convRef, contact.Name); err != nil {
		log.Printf("[MONITOR] poll_round=%s, stage=chat_verify, reason=verification_failed, error=%v", pollRound, err)
		return fmt.Errorf("验证聊天框失败: %v", err)
	}
	log.Printf("[MONITOR] poll_round=%s, stage=chat_verify, chat_open_verified=true", pollRound)

	// 步骤7: 输入并发送回复
	taskID := fmt.Sprintf("monitor_%d", time.Now().UnixNano())
	sendResult, err := ms.sendReply(convRef, replyContent, taskID)
	if err != nil {
		log.Printf("[MONITOR] poll_round=%s, stage=reply_send, reason=send_failed, error=%v", pollRound, err)
		return fmt.Errorf("发送回复失败: %v", err)
	}
	log.Printf("[MONITOR] poll_round=%s, stage=reply_send, reply_sent=true, confidence=%.2f", pollRound, sendResult.Confidence)

	// 步骤8: 更新session
	if err := ms.updateSessionAfterReply(contact.ID, replyContent, taskID, sendResult); err != nil {
		log.Printf("[MONITOR] poll_round=%s, stage=session_update, reason=reply_record_failed, error=%v", pollRound, err)
		return fmt.Errorf("更新回复记录失败: %v", err)
	}
	log.Printf("[MONITOR] poll_round=%s, stage=session_update, session_updated=true, final_state=success", pollRound)

	log.Printf("成功处理联系人 %s 的回复", contact.Name)
	return nil
}

// openContact 打开联系人（步骤2）
func (ms *MonitorService) openContact(contact ContactInfo) (protocol.ConversationRef, error) {
	log.Printf("步骤2: 打开联系人 %s", contact.Name)

	// 策略：列表优先，搜索兜底
	// 当前实现：列表优先 - 尝试直接聚焦到扫描到的联系人
	focusResult := ms.adapter.Focus(contact.Conversation)
	if focusResult.Status == adapter.StatusSuccess && focusResult.Confidence >= 0.8 {
		log.Printf("列表优先策略成功 (置信度: %.2f)", focusResult.Confidence)
		return contact.Conversation, nil
	}

	// 搜索兜底策略（需要adapter支持搜索功能）
	// TODO: 实现搜索框导航回退策略
	// 当前简化实现：返回错误
	log.Printf("列表优先策略失败，需要搜索兜底但adapter不支持 (置信度: %.2f, 错误: %s)",
		focusResult.Confidence, focusResult.Error)
	return protocol.ConversationRef{}, fmt.Errorf("无法打开联系人: %s (置信度: %.2f)", focusResult.Error, focusResult.Confidence)
}

// readNewMessages 读取新增消息（步骤3）
func (ms *MonitorService) readNewMessages(convRef protocol.ConversationRef, contactID string) ([]session.Message, error) {
	log.Printf("步骤3: 读取新增消息")

	// 读取最新消息
	messages, readResult := ms.adapter.Read(convRef, 20) // 读取最近20条
	if readResult.Status != adapter.StatusSuccess {
		return nil, fmt.Errorf("读取消息失败: %s", readResult.Error)
	}

	// 获取会话以过滤新消息
	session := ms.sessionMgr.Get(contactID)
	if session == nil {
		// 如果没有会话，所有消息都是新的
		return convertToSessionMessages(messages, contactID), nil
	}

	// 过滤出新消息（根据最后读取的消息ID）
	session.Mu.RLock()
	lastMessageID := session.LastMessageID
	session.Mu.RUnlock()

	return filterNewMessages(messages, lastMessageID, contactID), nil
}

// updateSessionWithMessages 更新会话消息（步骤4）
func (ms *MonitorService) updateSessionWithMessages(contactID string, messages []session.Message) error {
	log.Printf("步骤4: 更新会话")

	for _, msg := range messages {
		_, err := ms.sessionMgr.AddMessage(contactID, contactID, msg.Content, msg.Fingerprint, false)
		if err != nil {
			log.Printf("添加消息到会话失败: %v", err)
		}
	}

	// 标记为已读
	if len(messages) > 0 {
		lastMsgID := messages[len(messages)-1].ID
		if err := ms.sessionMgr.MarkAsRead(contactID, lastMsgID); err != nil {
			log.Printf("标记消息为已读失败: %v", err)
		}
	}

	return nil
}

// callRemoteAgent 调用远端agent（步骤5）
func (ms *MonitorService) callRemoteAgent(contactID string) (string, error) {
	log.Printf("步骤5: 调用远端agent")

	session := ms.sessionMgr.Get(contactID)
	if session == nil {
		return "", fmt.Errorf("会话不存在: %s", contactID)
	}

	// 构建上下文
	ctx, cancel := context.WithTimeout(ms.ctx, ms.operationTimeout)
	defer cancel()

	agentContext := AgentContext{
		ContactID:     contactID,
		ContactName:   session.ContactName,
		MessageHistory: session.MessageHistory,
		UnreadCount:   session.UnreadCount,
		LastReplyTime: session.LastReplyTime,
		Timestamp:     time.Now(),
	}

	// 调用远端agent
	reply, err := ms.agentClient.GetReply(ctx, agentContext)
	if err != nil {
		return "", fmt.Errorf("远端agent调用失败: %v", err)
	}

	// 设置待发送回复
	if err := ms.sessionMgr.SetPendingReply(contactID, reply); err != nil {
		log.Printf("设置待发送回复失败: %v", err)
	}

	return reply, nil
}

// verifyCurrentChat 验证当前聊天框（步骤6）
func (ms *MonitorService) verifyCurrentChat(convRef protocol.ConversationRef, contactName string) error {
	log.Printf("步骤6: 验证当前聊天框")

	// 简化实现：重新聚焦以确保在正确的聊天窗口
	focusResult := ms.adapter.Focus(convRef)
	if focusResult.Status != adapter.StatusSuccess {
		return fmt.Errorf("重新聚焦失败: %s", focusResult.Error)
	}

	log.Printf("聊天框验证成功 (置信度: %.2f)", focusResult.Confidence)
	return nil
}

// sendReply 发送回复（步骤7）
func (ms *MonitorService) sendReply(convRef protocol.ConversationRef, content, taskID string) (adapter.Result, error) {
	log.Printf("步骤7: 发送回复")

	// dry-run模式：不真正发送，模拟成功
	if ms.dryRun {
		log.Printf("[DRY-RUN] 模拟发送回复: content=%s (length=%d)", content, len(content))
		// 返回模拟的成功结果
		return adapter.Result{
			Status:     adapter.StatusSuccess,
			Confidence: 0.9,
			ElapsedMs:  100,
		}, nil
	}

	// 使用适配器发送消息
	sendResult := ms.adapter.Send(convRef, content, taskID)

	// 验证发送结果
	if sendResult.Status != adapter.StatusSuccess {
		return sendResult, fmt.Errorf("发送失败: %s (原因码: %s)", sendResult.Error, sendResult.ReasonCode)
	}

	log.Printf("发送成功 (置信度: %.2f, 耗时: %dms)", sendResult.Confidence, sendResult.ElapsedMs)

	// 可选：验证消息是否确实发送
	time.Sleep(500 * time.Millisecond)
	_, verifyResult := ms.adapter.Verify(convRef, content, 3*time.Second)
	if verifyResult.Status == adapter.StatusSuccess && verifyResult.Confidence >= 0.8 {
		log.Printf("消息验证成功 (置信度: %.2f)", verifyResult.Confidence)
	} else {
		log.Printf("消息验证置信度较低或失败: %s", verifyResult.Error)
	}

	return sendResult, nil
}

// updateSessionAfterReply 更新回复记录（步骤8）
func (ms *MonitorService) updateSessionAfterReply(contactID, content, taskID string, sendResult adapter.Result) error {
	log.Printf("步骤8: 更新会话")

	success := sendResult.Status == adapter.StatusSuccess
	confidence := sendResult.Confidence
	var errorMsg string
	if !success {
		errorMsg = sendResult.Error
	}

	// 使用taskID作为回复指纹（每个回复任务唯一）
	replyFingerprint := taskID
	_, err := ms.sessionMgr.AddReply(contactID, content, taskID, success, errorMsg, confidence, replyFingerprint)
	if err != nil {
		return fmt.Errorf("添加回复记录失败: %v", err)
	}

	// 清除待发送回复
	if err := ms.sessionMgr.ClearPendingReply(contactID); err != nil {
		log.Printf("清除待发送回复失败: %v", err)
	}

	return nil
}

// ContactInfo 联系人信息
type ContactInfo struct {
	ID           string
	Name         string
	UnreadCount  int
	Conversation protocol.ConversationRef
}

// estimateUnreadCount 估计未读消息数量（需要根据实际UI特征实现）
func estimateUnreadCount(conv protocol.ConversationRef) int {
	// 简化实现：返回0或1
	// 实际实现需要检测红点、未读计数等UI特征
	return 0
}

// convertToSessionMessages 将协议消息转换为会话消息
func convertToSessionMessages(messages []protocol.MessageObs, contactID string) []session.Message {
	var result []session.Message
	for _, msg := range messages {
		sessionMsg := session.Message{
			ID:          msg.MessageFingerprint,
			Sender:      contactID, // 简化：假设发送者就是联系人
			Content:     msg.NormalizedText,
			Timestamp:   msg.Timestamp,
			Fingerprint: msg.MessageFingerprint,
			IsOutgoing:  false,
		}
		result = append(result, sessionMsg)
	}
	return result
}

// filterNewMessages 过滤出新消息
func filterNewMessages(messages []protocol.MessageObs, lastMessageID, contactID string) []session.Message {
	var newMessages []session.Message
	foundLast := false

	if lastMessageID == "" {
		// 如果没有最后消息ID，所有消息都是新的
		return convertToSessionMessages(messages, contactID)
	}

	// 从最新到最旧遍历，直到找到最后处理的消息
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.MessageFingerprint == lastMessageID {
			foundLast = true
			break
		}
		sessionMsg := session.Message{
			ID:          msg.MessageFingerprint,
			Sender:      contactID,
			Content:     msg.NormalizedText,
			Timestamp:   msg.Timestamp,
			Fingerprint: msg.MessageFingerprint,
			IsOutgoing:  false,
		}
		newMessages = append(newMessages, sessionMsg)
	}

	// 如果没找到最后消息ID，所有消息都是新的
	if !foundLast {
		return convertToSessionMessages(messages, contactID)
	}

	// 反转顺序，使消息按时间顺序排列
	for i, j := 0, len(newMessages)-1; i < j; i, j = i+1, j-1 {
		newMessages[i], newMessages[j] = newMessages[j], newMessages[i]
	}

	return newMessages
}
