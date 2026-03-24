package handler

import (
	"log"

	"github.com/mazhiqiang666/GroupClaw-Desktop/internal/gateway/session"
	"github.com/mazhiqiang666/GroupClaw-Desktop/pkg/protocol"
)

// EventHandler 事件处理器
type EventHandler struct {
	registry *session.Registry
}

// NewEventHandler 创建事件处理器
func NewEventHandler(registry *session.Registry) *EventHandler {
	return &EventHandler{
		registry: registry,
	}
}

// Handle 处理事件
func (h *EventHandler) Handle(env *protocol.Envelope) error {
	switch env.PayloadType {
	case protocol.PayloadNewMessage:
		return h.handleNewMessage(env)
	case protocol.PayloadConvStatusChanged:
		return h.handleConvStatusChanged(env)
	case protocol.PayloadTaskProgress:
		return h.handleTaskProgress(env)
	case protocol.PayloadTaskCompleted:
		return h.handleTaskCompleted(env)
	case protocol.PayloadTaskFailed:
		return h.handleTaskFailed(env)
	default:
		log.Printf("Unknown event type: %s", env.PayloadType)
		return nil
	}
}

// handleNewMessage 处理新消息事件
func (h *EventHandler) handleNewMessage(env *protocol.Envelope) error {
	var payload protocol.NewMessagePayload
	if err := protocol.DecodeEnvelopePayload(env, &payload); err != nil {
		return err
	}

	log.Printf("New message: conv=%s, text=%s", payload.ConversationID, payload.Message.NormalizedText)

	// TODO: 转发到 LLM Orchestrator
	return nil
}

// handleConvStatusChanged 处理会话状态变更事件
func (h *EventHandler) handleConvStatusChanged(env *protocol.Envelope) error {
	var payload protocol.ConvStatusChangedPayload
	if err := protocol.DecodeEnvelopePayload(env, &payload); err != nil {
		return err
	}

	log.Printf("Conversation status changed: conv=%s, status=%s", payload.ConversationID, payload.NewStatus)
	return nil
}

// handleTaskProgress 处理任务进度事件
func (h *EventHandler) handleTaskProgress(env *protocol.Envelope) error {
	var payload protocol.TaskProgressPayload
	if err := protocol.DecodeEnvelopePayload(env, &payload); err != nil {
		return err
	}

	log.Printf("Task progress: task=%s, progress=%.0f%%, stage=%s", payload.TaskID, payload.Progress*100, payload.Stage)
	return nil
}

// handleTaskCompleted 处理任务完成事件
func (h *EventHandler) handleTaskCompleted(env *protocol.Envelope) error {
	var payload protocol.TaskCompletedPayload
	if err := protocol.DecodeEnvelopePayload(env, &payload); err != nil {
		return err
	}

	log.Printf("Task completed: task=%s, fingerprint=%s, delivery_state=%s", payload.TaskID, payload.ObservedMessageFingerprint, payload.DeliveryState)
	return nil
}

// handleTaskFailed 处理任务失败事件
func (h *EventHandler) handleTaskFailed(env *protocol.Envelope) error {
	var payload protocol.TaskFailedPayload
	if err := protocol.DecodeEnvelopePayload(env, &payload); err != nil {
		return err
	}

	log.Printf("Task failed: task=%s, code=%s, reason=%s", payload.TaskID, payload.ErrorCode, payload.ErrorReason)
	return nil
}
