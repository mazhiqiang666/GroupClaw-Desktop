package session

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/mazhiqiang666/GroupClaw-Desktop/pkg/protocol"
)

// Message 表示一条消息
type Message struct {
	ID          string    `json:"id"`
	Sender      string    `json:"sender"`
	Content     string    `json:"content"`
	Timestamp   time.Time `json:"timestamp"`
	Fingerprint string    `json:"fingerprint"` // 用于去重
	IsOutgoing  bool      `json:"is_outgoing"`
}

// ReplyRecord 回复记录
type ReplyRecord struct {
	ID           string    `json:"id"`
	Content      string    `json:"content"`
	Timestamp    time.Time `json:"timestamp"`
	TaskID       string    `json:"task_id"`
	Success      bool      `json:"success"`
	ErrorMessage string    `json:"error_message,omitempty"`
	Confidence   float64   `json:"confidence"`
}

// ChatSession 聊天会话
type ChatSession struct {
	Mu                 sync.RWMutex `json:"-"`
	ContactID          string       `json:"contact_id"`
	ContactName        string       `json:"contact_name"`
	CreatedAt          time.Time    `json:"created_at"`
	LastActivityAt     time.Time    `json:"last_activity_at"`
	LastReadTime       time.Time    `json:"last_read_time"`
	LastMessageID      string       `json:"last_message_id"`
	MessageHistory     []Message    `json:"message_history"`
	LastReplyTime      time.Time    `json:"last_reply_time"`
	LastReplyContent   string       `json:"last_reply_content"`
	ReplyHistory       []ReplyRecord `json:"reply_history"`
	PendingReply       string       `json:"pending_reply,omitempty"`
	UnreadCount        int          `json:"unread_count"`
	IsActive           bool         `json:"is_active"`
	ConversationRef    *protocol.ConversationRef `json:"conversation_ref,omitempty"`
	// 真实身份字段
	StableContactID    string `json:"stable_contact_id,omitempty"`   // 稳定联系人ID（基于多维度特征）
	DisplayContactName string `json:"display_contact_name,omitempty"` // 显示联系人名称（优先来源：左侧列表文本、聊天标题）
	// 去重字段
	LastProcessedIncomingFingerprint string `json:"last_processed_incoming_fingerprint"`
	LastSentReplyFingerprint        string `json:"last_sent_reply_fingerprint"`
}

// SessionManager 会话管理器
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*ChatSession // key: contact_id
	store    SessionStore
}

// SessionStore 会话存储接口
type SessionStore interface {
	Save(session *ChatSession) error
	Load(contactID string) (*ChatSession, error)
	Delete(contactID string) error
	List() ([]string, error)
}

// MemoryStore 内存存储实现
type MemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]*ChatSession
}

// NewSessionManager 创建新的会话管理器
func NewSessionManager(store SessionStore) *SessionManager {
	if store == nil {
		store = &MemoryStore{
			sessions: make(map[string]*ChatSession),
		}
	}

	return &SessionManager{
		sessions: make(map[string]*ChatSession),
		store:    store,
	}
}

// GetOrCreate 获取或创建会话
func (sm *SessionManager) GetOrCreate(contactID, contactName string) *ChatSession {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 先从内存中查找
	if session, exists := sm.sessions[contactID]; exists {
		session.LastActivityAt = time.Now()
		return session
	}

	// 尝试从存储加载
	if session, err := sm.store.Load(contactID); err == nil && session != nil {
		session.LastActivityAt = time.Now()
		sm.sessions[contactID] = session
		return session
	}

	// 创建新会话
	session := &ChatSession{
		ContactID:       contactID,
		ContactName:     contactName,
		CreatedAt:       time.Now(),
		LastActivityAt:  time.Now(),
		LastReadTime:    time.Time{},
		LastMessageID:   "",
		MessageHistory:  []Message{},
		LastReplyTime:   time.Time{},
		LastReplyContent: "",
		ReplyHistory:    []ReplyRecord{},
		PendingReply:    "",
		UnreadCount:     0,
		IsActive:        false,
		ConversationRef: nil,
		// 真实身份字段初始化（后续可更新）
		StableContactID:    contactID,   // 初始使用contactID，后续可更新为更稳定的ID
		DisplayContactName: contactName, // 初始使用contactName，后续可更新为真实显示名称
		// 去重字段初始化
		LastProcessedIncomingFingerprint: "",
		LastSentReplyFingerprint:        "",
	}

	sm.sessions[contactID] = session

	// 异步保存到存储
	go func() {
		_ = sm.store.Save(session)
	}()

	return session
}

// Get 获取会话（如果不存在返回nil）
func (sm *SessionManager) Get(contactID string) *ChatSession {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return sm.sessions[contactID]
}

// Save 保存会话到存储
func (sm *SessionManager) Save(session *ChatSession) error {
	session.Mu.Lock()
	defer session.Mu.Unlock()

	session.LastActivityAt = time.Now()

	// 确保会话在内存中
	sm.mu.Lock()
	sm.sessions[session.ContactID] = session
	sm.mu.Unlock()

	// 保存到存储
	return sm.store.Save(session)
}

// AddMessage 添加消息到会话
func (sm *SessionManager) AddMessage(contactID, sender, content, fingerprint string, isOutgoing bool) (*Message, error) {
	session := sm.GetOrCreate(contactID, sender)
	if session == nil {
		return nil, fmt.Errorf("failed to get or create session for %s", contactID)
	}

	session.Mu.Lock()
	defer session.Mu.Unlock()

	// 检查是否重复消息
	for _, msg := range session.MessageHistory {
		if msg.Fingerprint == fingerprint {
			return &msg, nil // 重复消息
		}
	}

	msg := Message{
		ID:          fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		Sender:      sender,
		Content:     content,
		Timestamp:   time.Now(),
		Fingerprint: fingerprint,
		IsOutgoing:  isOutgoing,
	}

	session.MessageHistory = append(session.MessageHistory, msg)
	session.LastMessageID = msg.ID

	if !isOutgoing {
		session.UnreadCount++
		// 更新最后处理的消息指纹
		session.LastProcessedIncomingFingerprint = fingerprint
	}

	session.LastActivityAt = time.Now()

	// 限制历史记录大小
	if len(session.MessageHistory) > 100 {
		session.MessageHistory = session.MessageHistory[1:]
	}

	// 异步保存
	go func() {
		_ = sm.Save(session)
	}()

	return &msg, nil
}

// MarkAsRead 标记消息为已读
func (sm *SessionManager) MarkAsRead(contactID string, lastMessageID string) error {
	session := sm.Get(contactID)
	if session == nil {
		return fmt.Errorf("session not found for %s", contactID)
	}

	session.Mu.Lock()
	defer session.Mu.Unlock()

	session.LastReadTime = time.Now()
	session.LastMessageID = lastMessageID
	session.UnreadCount = 0
	session.LastActivityAt = time.Now()

	go func() {
		_ = sm.Save(session)
	}()

	return nil
}

// AddReply 添加回复记录
func (sm *SessionManager) AddReply(contactID, content, taskID string, success bool, errorMsg string, confidence float64, replyFingerprint string) (*ReplyRecord, error) {
	session := sm.Get(contactID)
	if session == nil {
		return nil, fmt.Errorf("session not found for %s", contactID)
	}

	session.Mu.Lock()
	defer session.Mu.Unlock()

	record := ReplyRecord{
		ID:           fmt.Sprintf("reply_%d", time.Now().UnixNano()),
		Content:      content,
		Timestamp:    time.Now(),
		TaskID:       taskID,
		Success:      success,
		ErrorMessage: errorMsg,
		Confidence:   confidence,
	}

	session.ReplyHistory = append(session.ReplyHistory, record)
	session.LastReplyTime = record.Timestamp
	session.LastReplyContent = content
	session.PendingReply = ""
	session.LastActivityAt = time.Now()
	// 更新最后发送的回复指纹
	if success && replyFingerprint != "" {
		session.LastSentReplyFingerprint = replyFingerprint
	}

	// 限制历史记录大小
	if len(session.ReplyHistory) > 50 {
		session.ReplyHistory = session.ReplyHistory[1:]
	}

	go func() {
		_ = sm.Save(session)
	}()

	return &record, nil
}

// SetPendingReply 设置待发送回复
func (sm *SessionManager) SetPendingReply(contactID, content string) error {
	session := sm.Get(contactID)
	if session == nil {
		return fmt.Errorf("session not found for %s", contactID)
	}

	session.Mu.Lock()
	defer session.Mu.Unlock()

	session.PendingReply = content
	session.LastActivityAt = time.Now()

	return nil
}

// ClearPendingReply 清除待发送回复
func (sm *SessionManager) ClearPendingReply(contactID string) error {
	session := sm.Get(contactID)
	if session == nil {
		return fmt.Errorf("session not found for %s", contactID)
	}

	session.Mu.Lock()
	defer session.Mu.Unlock()

	session.PendingReply = ""
	session.LastActivityAt = time.Now()

	return nil
}

// SetConversationRef 设置会话引用
func (sm *SessionManager) SetConversationRef(contactID string, convRef *protocol.ConversationRef) error {
	session := sm.GetOrCreate(contactID, convRef.DisplayName)
	if session == nil {
		return fmt.Errorf("failed to get or create session for %s", contactID)
	}

	session.Mu.Lock()
	defer session.Mu.Unlock()

	session.ConversationRef = convRef
	session.IsActive = true
	session.LastActivityAt = time.Now()

	go func() {
		_ = sm.Save(session)
	}()

	return nil
}

// GetActiveSessions 获取活跃会话列表
func (sm *SessionManager) GetActiveSessions() []*ChatSession {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var activeSessions []*ChatSession
	for _, session := range sm.sessions {
		if session.IsActive {
			activeSessions = append(activeSessions, session)
		}
	}
	return activeSessions
}

// GetSessionsWithUnread 获取有未读消息的会话
func (sm *SessionManager) GetSessionsWithUnread() []*ChatSession {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var unreadSessions []*ChatSession
	for _, session := range sm.sessions {
		session.Mu.RLock()
		hasUnread := session.UnreadCount > 0
		session.Mu.RUnlock()

		if hasUnread {
			unreadSessions = append(unreadSessions, session)
		}
	}
	return unreadSessions
}

// MemoryStore 实现

// Save 保存会话到内存存储
func (ms *MemoryStore) Save(session *ChatSession) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// 创建副本以避免竞态
	sessionCopy := *session
	ms.sessions[session.ContactID] = &sessionCopy
	return nil
}

// Load 从内存存储加载会话
func (ms *MemoryStore) Load(contactID string) (*ChatSession, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	session, exists := ms.sessions[contactID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", contactID)
	}

	// 返回副本
	sessionCopy := *session
	return &sessionCopy, nil
}

// Delete 从内存存储删除会话
func (ms *MemoryStore) Delete(contactID string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	delete(ms.sessions, contactID)
	return nil
}

// List 列出所有会话ID
func (ms *MemoryStore) List() ([]string, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	var ids []string
	for id := range ms.sessions {
		ids = append(ids, id)
	}
	return ids, nil
}

// FileStore 文件存储实现
type FileStore struct {
	path string
	mu   sync.RWMutex
}

// NewFileStore 创建文件存储
func NewFileStore(path string) *FileStore {
	return &FileStore{
		path: path,
	}
}

// Save 保存会话到文件
func (fs *FileStore) Save(session *ChatSession) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// 加载现有数据
	sessions := make(map[string]*ChatSession)
	if data, err := os.ReadFile(fs.path); err == nil {
		json.Unmarshal(data, &sessions)
	}

	// 更新会话
	sessions[session.ContactID] = session

	// 写回文件
	data, err := json.MarshalIndent(sessions, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(fs.path, data, 0644)
}

// Load 从文件加载会话
func (fs *FileStore) Load(contactID string) (*ChatSession, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	data, err := os.ReadFile(fs.path)
	if err != nil {
		return nil, err
	}

	var sessions map[string]*ChatSession
	if err := json.Unmarshal(data, &sessions); err != nil {
		return nil, err
	}

	session, exists := sessions[contactID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", contactID)
	}

	return session, nil
}

// Delete 从文件删除会话
func (fs *FileStore) Delete(contactID string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// 加载现有数据
	sessions := make(map[string]*ChatSession)
	if data, err := os.ReadFile(fs.path); err == nil {
		json.Unmarshal(data, &sessions)
	}

	// 删除会话
	delete(sessions, contactID)

	// 写回文件
	data, err := json.MarshalIndent(sessions, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(fs.path, data, 0644)
}

// List 列出所有会话ID
func (fs *FileStore) List() ([]string, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	data, err := os.ReadFile(fs.path)
	if err != nil {
		return nil, err
	}

	var sessions map[string]*ChatSession
	if err := json.Unmarshal(data, &sessions); err != nil {
		return nil, err
	}

	var ids []string
	for id := range sessions {
		ids = append(ids, id)
	}
	return ids, nil
}
