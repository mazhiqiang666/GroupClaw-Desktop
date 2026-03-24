package unit

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/mazhiqiang666/GroupClaw-Desktop/pkg/protocol"
)

func TestProtocolSerialization(t *testing.T) {
	// 构造 NewMessage 事件载荷
	payload := protocol.NewMessagePayload{
		ConversationID: "conv_001",
		Message: protocol.MessageObs{
			MessageID:          "msg_001",
			ConversationID:     "conv_001",
			SenderSide:         "customer",
			NormalizedText:     "你好",
			Timestamp:          time.Now(),
			ObservedAt:         time.Now(),
			MessageFingerprint: "fp_abc123",
		},
	}

	// 创建信封
	env, err := protocol.NewEnvelope(protocol.KindEvent, protocol.PayloadNewMessage, payload)
	if err != nil {
		t.Fatalf("Failed to create envelope: %v", err)
	}

	// 序列化
	data, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// 反序列化
	var decoded protocol.Envelope
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// 验证
	if decoded.Kind != protocol.KindEvent {
		t.Errorf("Kind mismatch: got %v, want %v", decoded.Kind, protocol.KindEvent)
	}

	if decoded.PayloadType != protocol.PayloadNewMessage {
		t.Errorf("PayloadType mismatch: got %v, want %v", decoded.PayloadType, protocol.PayloadNewMessage)
	}
}

func TestDecodePayload(t *testing.T) {
	// 构造载荷
	payload := protocol.ReplyExecutePayload{
		ConversationID: "conv_001",
		ReplyContent:   "测试回复",
	}

	// 创建信封
	env, err := protocol.NewEnvelope(protocol.KindCommand, protocol.PayloadReplyExecute, payload)
	if err != nil {
		t.Fatalf("Failed to create envelope: %v", err)
	}

	// 解码载荷（使用运行时 DecodeEnvelopePayload）
	var decoded protocol.ReplyExecutePayload
	err = protocol.DecodeEnvelopePayload(env, &decoded)
	if err != nil {
		t.Fatalf("DecodeEnvelopePayload failed: %v", err)
	}

	// 验证
	if decoded.ConversationID != "conv_001" {
		t.Errorf("ConversationID mismatch: got %v, want %v", decoded.ConversationID, "conv_001")
	}

	if decoded.ReplyContent != "测试回复" {
		t.Errorf("ReplyContent mismatch: got %v, want %v", decoded.ReplyContent, "测试回复")
	}
}

func TestPayloadRoundTrip(t *testing.T) {
	// 测试 NewMessagePayload 的完整序列化/反序列化
	original := protocol.NewMessagePayload{
		ConversationID: "conv_001",
		Message: protocol.MessageObs{
			MessageID:          "msg_001",
			ConversationID:     "conv_001",
			SenderSide:         "customer",
			NormalizedText:     "你好，世界",
			Timestamp:          time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			ObservedAt:         time.Date(2024, 1, 1, 12, 0, 1, 0, time.UTC),
			MessageFingerprint: "fp_test123",
			NeighborFingerprint: "fp_neighbor456",
		},
	}

	// 创建信封
	env, err := protocol.NewEnvelope(protocol.KindEvent, protocol.PayloadNewMessage, original)
	if err != nil {
		t.Fatalf("Failed to create envelope: %v", err)
	}

	// 序列化信封
	data, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("Marshal envelope failed: %v", err)
	}

	// 反序列化信封
	var decodedEnv protocol.Envelope
	err = json.Unmarshal(data, &decodedEnv)
	if err != nil {
		t.Fatalf("Unmarshal envelope failed: %v", err)
	}

	// 解码载荷
	var decoded protocol.NewMessagePayload
	err = protocol.DecodeEnvelopePayload(&decodedEnv, &decoded)
	if err != nil {
		t.Fatalf("DecodeEnvelopePayload failed: %v", err)
	}

	// 验证完整轮转
	if decoded.ConversationID != original.ConversationID {
		t.Errorf("ConversationID mismatch: got %v, want %v", decoded.ConversationID, original.ConversationID)
	}

	if decoded.Message.MessageID != original.Message.MessageID {
		t.Errorf("MessageID mismatch: got %v, want %v", decoded.Message.MessageID, original.Message.MessageID)
	}

	if decoded.Message.NormalizedText != original.Message.NormalizedText {
		t.Errorf("NormalizedText mismatch: got %v, want %v", decoded.Message.NormalizedText, original.Message.NormalizedText)
	}

	if decoded.Message.MessageFingerprint != original.Message.MessageFingerprint {
		t.Errorf("MessageFingerprint mismatch: got %v, want %v", decoded.Message.MessageFingerprint, original.Message.MessageFingerprint)
	}
}
