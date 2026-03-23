package protocol

import "time"

// ============ Command Payloads (Server → Agent) ============

// ReplyExecutePayload reply.execute 命令载荷
type ReplyExecutePayload struct {
	ConversationID string `json:"conversation_id"`
	ReplyContent   string `json:"reply_content"`
}

func (p ReplyExecutePayload) Type() PayloadType { return PayloadReplyExecute }

// ConvModeSetPayload conversation.mode.set 命令载荷
type ConvModeSetPayload struct {
	ConversationID string `json:"conversation_id"`
	Mode           string `json:"mode"` // auto | review | manual
}

func (p ConvModeSetPayload) Type() PayloadType { return PayloadConvModeSet }

// DiagnosticCapturePayload diagnostic.capture 命令载荷
type DiagnosticCapturePayload struct {
	ConversationID string `json:"conversation_id"`
	CaptureType    string `json:"capture_type"` // screenshot | logs | ui_tree
}

func (p DiagnosticCapturePayload) Type() PayloadType { return PayloadDiagnosticCapture }

// ============ Event Payloads (Agent → Server) ============

// NewMessagePayload conversation.new_message 事件载荷
type NewMessagePayload struct {
	ConversationID string     `json:"conversation_id"`
	Message        MessageObs `json:"message"`
}

func (p NewMessagePayload) Type() PayloadType { return PayloadNewMessage }

// ConvStatusChangedPayload conversation.status_changed 事件载荷
type ConvStatusChangedPayload struct {
	ConversationID string `json:"conversation_id"`
	OldStatus      string `json:"old_status"`
	NewStatus      string `json:"new_status"`
}

func (p ConvStatusChangedPayload) Type() PayloadType { return PayloadConvStatusChanged }

// TaskProgressPayload task.progress 事件载荷
type TaskProgressPayload struct {
	TaskID   string  `json:"task_id"`
	Progress float64 `json:"progress"` // 0.0 - 1.0
	Message  string  `json:"message"`  // 进度描述
	Stage    string  `json:"stage"`    // 阶段名称
}

func (p TaskProgressPayload) Type() PayloadType { return PayloadTaskProgress }

// TaskCompletedPayload task.completed 事件载荷
type TaskCompletedPayload struct {
	TaskID                     string  `json:"task_id"`
	ObservedMessageFingerprint string  `json:"observed_message_fingerprint"`
	VerificationConfidence     float64 `json:"verification_confidence"`
	DeliveryState              string  `json:"delivery_state"` // delivered | verified
}

func (p TaskCompletedPayload) Type() PayloadType { return PayloadTaskCompleted }

// TaskFailedPayload task.failed 事件载荷
type TaskFailedPayload struct {
	TaskID      string `json:"task_id"`
	ErrorCode   string `json:"error_code"`
	ErrorReason string `json:"error_reason"`
}

func (p TaskFailedPayload) Type() PayloadType { return PayloadTaskFailed }

// ============ Audit Payloads (Agent → Server) ============

// ActionAuditPayload action 审计载荷
type ActionAuditPayload struct {
	TaskID     string    `json:"task_id"`
	ActionType string    `json:"action_type"`
	ActionData string    `json:"action_data"`
	Timestamp  time.Time `json:"timestamp"`
}

func (p ActionAuditPayload) Type() PayloadType { return PayloadActionAudit }

// ============ Shared Data Models ============

// MessageObs 观测消息模型
type MessageObs struct {
	MessageID          string    `json:"message_id,omitempty"`
	ConversationID     string    `json:"conversation_id"`
	SenderSide         string    `json:"sender_side"`
	NormalizedText     string    `json:"normalized_text"`
	Timestamp          time.Time `json:"timestamp"`
	ObservedAt         time.Time `json:"observed_at"`
	MessageFingerprint string    `json:"message_fingerprint"`
	NeighborFingerprint string   `json:"neighbor_fingerprint"`
}

// AppInstanceRef 应用实例引用
type AppInstanceRef struct {
	AppID      string `json:"app_id"`
	InstanceID string `json:"instance_id"`
}

// ConversationRef 运行时会话引用
type ConversationRef struct {
	HostWindowHandle        uintptr        `json:"host_window_handle"` // host window hint，不是唯一标识
	AppInstance             AppInstanceRef `json:"app_instance"`
	DisplayName             string         `json:"display_name"`
	PreviewText             string         `json:"preview_text"`
	AvatarHash              string         `json:"avatar_hash"`
	ListPosition            int            `json:"list_position"`            // 列表位置（用于 neighborhood 匹配）
	ListNeighborhoodHint    []string       `json:"list_neighborhood_hint"`   // 列表邻域提示
	RecentMessageFingerprint string        `json:"recent_message_fingerprint"` // 最近消息指纹
}

// ConversationIdentity 逻辑会话身份
type ConversationIdentity struct {
	IdentityHash string    `json:"identity_hash"` // 主键（基于多维度特征生成）
	TenantID     string    `json:"tenant_id"`
	DeviceID     string    `json:"device_id"`
	DisplayName  string    `json:"display_name"`
	AvatarHash   string    `json:"avatar_hash"`
	Status       string    `json:"status"` // active | inactive | closed
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
