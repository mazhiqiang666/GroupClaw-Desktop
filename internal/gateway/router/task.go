package router

import (
	"log"

	"github.com/yourorg/auto-customer-service/internal/gateway/session"
	"github.com/yourorg/auto-customer-service/pkg/protocol"
)

// TaskRouter 任务路由器
type TaskRouter struct {
	registry *session.Registry
}

// NewTaskRouter 创建任务路由器
func NewTaskRouter(registry *session.Registry) *TaskRouter {
	return &TaskRouter{
		registry: registry,
	}
}

// RouteReplyExecute 路由 reply.execute 命令
func (r *TaskRouter) RouteReplyExecute(payload protocol.ReplyExecutePayload, targetSessionID string) error {
	env, err := protocol.NewEnvelope(
		protocol.KindCommand,
		protocol.PayloadReplyExecute,
		payload,
	)
	if err != nil {
		return err
	}

	env.DeviceID = targetSessionID
	env.TaskID = protocol.GenerateTaskID()

	session, exists := r.registry.Get(targetSessionID)
	if !exists {
		log.Printf("Session not found: %s", targetSessionID)
		return nil
	}

	if err := session.Conn.Send(env); err != nil {
		log.Printf("Failed to route reply execute: %v", err)
		return err
	}

	log.Printf("Reply execute routed: task=%s, session=%s", env.TaskID, targetSessionID)
	return nil
}

// RouteConvModeSet 路由 conversation.mode.set 命令
func (r *TaskRouter) RouteConvModeSet(payload protocol.ConvModeSetPayload, targetSessionID string) error {
	env, err := protocol.NewEnvelope(
		protocol.KindCommand,
		protocol.PayloadConvModeSet,
		payload,
	)
	if err != nil {
		return err
	}

	env.DeviceID = targetSessionID

	session, exists := r.registry.Get(targetSessionID)
	if !exists {
		log.Printf("Session not found: %s", targetSessionID)
		return nil
	}

	if err := session.Conn.Send(env); err != nil {
		log.Printf("Failed to route conv mode set: %v", err)
		return err
	}

	log.Printf("Conversation mode set routed: conv=%s, mode=%s, session=%s", payload.ConversationID, payload.Mode, targetSessionID)
	return nil
}
