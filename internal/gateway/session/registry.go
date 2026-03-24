package session

import (
	"sync"
	"time"

	"github.com/mazhiqiang666/GroupClaw-Desktop/pkg/protocol"
)

// Session 会话连接
type Session struct {
	ID         string
	DeviceID   string
	TenantID   string
	Conn       Connection
	LastSeen   time.Time
	Metadata   map[string]string
}

// Connection 连接接口
type Connection interface {
	Send(env *protocol.Envelope) error
	Close() error
	IsClosed() bool
}

// Registry 会话注册表
type Registry struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

// NewRegistry 创建注册表
func NewRegistry() *Registry {
	return &Registry{
		sessions: make(map[string]*Session),
	}
}

// Register 注册会话
func (r *Registry) Register(session *Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.sessions[session.ID] = session
	return nil
}

// Unregister 注销会话
func (r *Registry) Unregister(sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.sessions, sessionID)
	return nil
}

// Get 获取会话
func (r *Registry) Get(sessionID string) (*Session, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	session, exists := r.sessions[sessionID]
	return session, exists
}

// ListByTenant 按租户列出会话
func (r *Registry) ListByTenant(tenantID string) []*Session {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var sessions []*Session
	for _, session := range r.sessions {
		if session.TenantID == tenantID {
			sessions = append(sessions, session)
		}
	}
	return sessions
}

// ListAll 列出所有会话
func (r *Registry) ListAll() []*Session {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var sessions []*Session
	for _, session := range r.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}

// UpdateLastSeen 更新最后活跃时间
func (r *Registry) UpdateLastSeen(sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if session, exists := r.sessions[sessionID]; exists {
		session.LastSeen = time.Now()
	}
	return nil
}
