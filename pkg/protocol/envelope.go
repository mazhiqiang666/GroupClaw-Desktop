package protocol

import (
	"encoding/json"
	"fmt"
	"time"
)

// MessageKind 消息类型（方向）
type MessageKind string

const (
	KindCommand MessageKind = "command" // Server → Agent (下行)
	KindEvent   MessageKind = "event"   // Agent → Server (上行)
	KindAudit   MessageKind = "audit"   // Agent → Server (上行)
)

// PayloadType 载荷类型
type PayloadType string

const (
	// Command payloads (Server → Agent)
	PayloadReplyExecute      PayloadType = "reply.execute"
	PayloadConvModeSet       PayloadType = "conversation.mode.set"
	PayloadDiagnosticCapture PayloadType = "diagnostic.capture"

	// Event payloads (Agent → Server)
	PayloadNewMessage        PayloadType = "conversation.new_message"
	PayloadConvStatusChanged PayloadType = "conversation.status_changed"
	PayloadTaskProgress      PayloadType = "task.progress"
	PayloadTaskCompleted     PayloadType = "task.completed"
	PayloadTaskFailed        PayloadType = "task.failed"

	// Audit payloads (Agent → Server)
	PayloadActionAudit PayloadType = "action"
)

// Envelope 消息信封
type Envelope struct {
	MessageID     string          `json:"message_id"`
	Kind          MessageKind     `json:"kind"`
	PayloadType   PayloadType     `json:"payload_type"`
	TenantID      string          `json:"tenant_id"`
	DeviceID      string          `json:"device_id"`
	TaskID        string          `json:"task_id,omitempty"`
	TraceID       string          `json:"trace_id"`
	Timestamp     time.Time       `json:"timestamp"`
	SchemaVersion string          `json:"schema_version"`
	Payload       json.RawMessage `json:"payload"` // JSON encoded payload
}

// Payload 接口 - 所有载荷必须实现
type Payload interface {
	Type() PayloadType
}

// DecodeEnvelopePayload 解码信封中的载荷到具体类型
func DecodeEnvelopePayload(env *Envelope, payload Payload) error {
	if env.PayloadType != payload.Type() {
		return fmt.Errorf("payload type mismatch: got %s, want %s", env.PayloadType, payload.Type())
	}
	return json.Unmarshal(env.Payload, payload)
}

// DecodePayload 解码载荷到具体类型（辅助函数）
func DecodePayload(payloadType PayloadType, data []byte, payload Payload) error {
	if payloadType != payload.Type() {
		return fmt.Errorf("payload type mismatch: got %s, want %s", payloadType, payload.Type())
	}
	return json.Unmarshal(data, payload)
}

// EncodePayload 编码载荷
func EncodePayload(payload Payload) ([]byte, error) {
	return json.Marshal(payload)
}

// NewEnvelope 创建消息信封
func NewEnvelope(kind MessageKind, payloadType PayloadType, payload Payload) (*Envelope, error) {
	payloadBytes, err := EncodePayload(payload)
	if err != nil {
		return nil, err
	}

	return &Envelope{
		MessageID:     generateMessageID(),
		Kind:          kind,
		PayloadType:   payloadType,
		Timestamp:     time.Now(),
		SchemaVersion: "1.0.0",
		Payload:       payloadBytes,
	}, nil
}
