package idempotency

import "time"

// Store 幂等存储接口
type Store interface {
	CreateRecord(record Record) error
	GetRecord(taskID string) (*Record, error)
	UpdateRecord(taskID string, updates RecordUpdate) error
	CheckDuplicate(dedupeKey string) (*Record, error)
	ListRecords(conversationID string) ([]Record, error)
}

// Record 幂等记录
type Record struct {
	TaskID        string    `json:"task_id"`
	DedupeKey     string    `json:"dedupe_key"`
	Conversation  string    `json:"conversation"`
	Content       string    `json:"content"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	MessageID     string    `json:"message_id,omitempty"`
	Fingerprint   string    `json:"fingerprint"`     // 消息指纹
	VerifyStatus  string    `json:"verify_status"`
	VerifyCount   int       `json:"verify_count"`
	ParentTaskID  string    `json:"parent_task_id"`  // 父任务 ID（重发时）
}

// RecordUpdate 记录更新参数（强类型）
type RecordUpdate struct {
	Status        *string
	MessageID     *string
	Fingerprint   *string
	VerifyStatus  *string
	VerifyCount   *int
}

// NewRecordUpdate 创建更新参数
func NewRecordUpdate() RecordUpdate {
	return RecordUpdate{}
}

// WithStatus 设置状态
func (u RecordUpdate) WithStatus(status string) RecordUpdate {
	u.Status = &status
	return u
}

// WithMessageID 设置消息 ID
func (u RecordUpdate) WithMessageID(messageID string) RecordUpdate {
	u.MessageID = &messageID
	return u
}

// WithFingerprint 设置指纹
func (u RecordUpdate) WithFingerprint(fingerprint string) RecordUpdate {
	u.Fingerprint = &fingerprint
	return u
}

// WithVerifyStatus 设置验证状态
func (u RecordUpdate) WithVerifyStatus(verifyStatus string) RecordUpdate {
	u.VerifyStatus = &verifyStatus
	return u
}

// WithVerifyCount 设置验证次数
func (u RecordUpdate) WithVerifyCount(count int) RecordUpdate {
	u.VerifyCount = &count
	return u
}
