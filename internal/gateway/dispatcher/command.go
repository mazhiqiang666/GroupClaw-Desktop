package dispatcher

import (
	"log"

	"github.com/yourorg/auto-customer-service/internal/gateway/session"
	"github.com/yourorg/auto-customer-service/pkg/protocol"
)

// CommandDispatcher 命令分发器
type CommandDispatcher struct {
	registry *session.Registry
}

// NewCommandDispatcher 创建命令分发器
func NewCommandDispatcher(registry *session.Registry) *CommandDispatcher {
	return &CommandDispatcher{
		registry: registry,
	}
}

// Dispatch 分发命令
func (d *CommandDispatcher) Dispatch(env *protocol.Envelope, targetSessionID string) error {
	session, exists := d.registry.Get(targetSessionID)
	if !exists {
		log.Printf("Session not found: %s", targetSessionID)
		return nil
	}

	// 发送命令到 Agent
	if err := session.Conn.Send(env); err != nil {
		log.Printf("Failed to send command: %v", err)
		return err
	}

	log.Printf("Command dispatched: type=%s, session=%s", env.PayloadType, targetSessionID)
	return nil
}

// Broadcast 广播命令到所有会话
func (d *CommandDispatcher) Broadcast(env *protocol.Envelope) error {
	sessions := d.registry.ListAll()

	for _, session := range sessions {
		if err := session.Conn.Send(env); err != nil {
			log.Printf("Failed to broadcast to session %s: %v", session.ID, err)
			continue
		}
	}

	log.Printf("Command broadcast: type=%s, sessions=%d", env.PayloadType, len(sessions))
	return nil
}
