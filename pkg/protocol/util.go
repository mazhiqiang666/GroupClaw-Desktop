package protocol

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// generateMessageID 生成消息 ID
func generateMessageID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("msg_%s", hex.EncodeToString(b))
}

// GenerateTaskID 生成任务 ID
func GenerateTaskID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("task_%s", hex.EncodeToString(b))
}

// GenerateConversationID 生成会话 ID
func GenerateConversationID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("conv_%s", hex.EncodeToString(b))
}

// GenerateSessionID 生成会话连接 ID（独立于 ConversationID）
func GenerateSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("sess_%s", hex.EncodeToString(b))
}
