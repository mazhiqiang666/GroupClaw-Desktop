package state

import (
	"sync"
	"time"
)

// ConversationMode 会话模式
type ConversationMode string

const (
	ModeAuto   ConversationMode = "auto"   // 自动模式
	ModeReview ConversationMode = "review" // 审核模式
	ModeManual ConversationMode = "manual" // 手动模式
)

// ConversationModeState 会话模式状态
type ConversationModeState struct {
	ConversationID string           `json:"conversation_id"`
	Mode           ConversationMode `json:"mode"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

// TaskReviewState 任务审核状态
type TaskReviewState string

const (
	ReviewRequired      TaskReviewState = "review_required"
	AwaitingReview      TaskReviewState = "awaiting_review_approval"
	ReviewRejected      TaskReviewState = "review_rejected"
	Approved            TaskReviewState = "approved"
)

// TaskReviewStateInfo 任务审核状态信息
type TaskReviewStateInfo struct {
	TaskID        string          `json:"task_id"`
	ReviewState   TaskReviewState `json:"review_state"`
	ReviewerID    string          `json:"reviewer_id,omitempty"`
	ReviewedAt    time.Time       `json:"reviewed_at,omitempty"`
	Comment       string          `json:"comment,omitempty"`
}

// MessageStatus 消息状态
type MessageStatus string

const (
	StatusIdle                MessageStatus = "idle"
	StatusDetecting           MessageStatus = "detecting"
	StatusWindowValidating    MessageStatus = "window_validating"
	StatusConvValidating      MessageStatus = "conversation_validating"
	StatusReading             MessageStatus = "reading"
	StatusInputValidating     MessageStatus = "input_validating"
	StatusSending             MessageStatus = "sending"
	StatusSentUnverified      MessageStatus = "sent_unverified"
	StatusVerified            MessageStatus = "verified"
	StatusVerificationTimeout MessageStatus = "verification_timeout"
	StatusUnknownDelivery     MessageStatus = "unknown_delivery_state"
	StatusRetrying            MessageStatus = "retrying"
	StatusManualTakeover      MessageStatus = "manual_takeover"
	StatusBlockedByPolicy     MessageStatus = "blocked_by_policy"
)

// ReviewState 审核状态
type ReviewState struct {
	ConversationID string
	Status         ReviewStatus
	Reason         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ReviewStatus 审核状态枚举
type ReviewStatus string

const (
	ReviewPending   ReviewStatus = "pending"   // 待审核
	ReviewApproved  ReviewStatus = "approved"  // 已批准
	ReviewRejectedStatus ReviewStatus = "rejected"  // 已拒绝
	ReviewEscalated ReviewStatus = "escalated" // 已升级
)

// ReviewManager 审核状态管理器
type ReviewManager struct {
	states map[string]*ReviewState
	mu     sync.RWMutex
}

// NewReviewManager 创建审核状态管理器
func NewReviewManager() *ReviewManager {
	return &ReviewManager{
		states: make(map[string]*ReviewState),
	}
}

// GetState 获取会话的审核状态
func (m *ReviewManager) GetState(conversationID string) (*ReviewState, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, exists := m.states[conversationID]
	return state, exists
}

// SetState 设置会话的审核状态
func (m *ReviewManager) SetState(conversationID string, status ReviewStatus, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, exists := m.states[conversationID]
	if !exists {
		state = &ReviewState{
			ConversationID: conversationID,
			CreatedAt:      time.Now(),
		}
		m.states[conversationID] = state
	}

	state.Status = status
	state.Reason = reason
	state.UpdatedAt = time.Now()
	return nil
}

// ListByStatus 按状态列出审核会话
func (m *ReviewManager) ListByStatus(status ReviewStatus) []ReviewState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []ReviewState
	for _, state := range m.states {
		if state.Status == status {
			result = append(result, *state)
		}
	}
	return result
}

// RemoveState 移除审核状态
func (m *ReviewManager) RemoveState(conversationID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.states, conversationID)
}

// ReviewTask 审核任务
type ReviewTask struct {
	TaskID         string
	ConversationID string
	ActionType     string
	ActionData     string
	Reason         string
	Status         ReviewStatus
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ReviewTaskManager 审核任务管理器
type ReviewTaskManager struct {
	tasks map[string]*ReviewTask
	mu    sync.RWMutex
}

// NewReviewTaskManager 创建审核任务管理器
func NewReviewTaskManager() *ReviewTaskManager {
	return &ReviewTaskManager{
		tasks: make(map[string]*ReviewTask),
	}
}

// CreateTask 创建审核任务
func (m *ReviewTaskManager) CreateTask(taskID, conversationID, actionType, actionData, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.tasks[taskID]; exists {
		return nil // 已存在
	}

	m.tasks[taskID] = &ReviewTask{
		TaskID:         taskID,
		ConversationID: conversationID,
		ActionType:     actionType,
		ActionData:     actionData,
		Reason:         reason,
		Status:         ReviewPending,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	return nil
}

// GetTask 获取审核任务
func (m *ReviewTaskManager) GetTask(taskID string) (*ReviewTask, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	task, exists := m.tasks[taskID]
	return task, exists
}

// UpdateTaskStatus 更新任务状态
func (m *ReviewTaskManager) UpdateTaskStatus(taskID string, status ReviewStatus, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return nil
	}

	task.Status = status
	task.Reason = reason
	task.UpdatedAt = time.Now()
	return nil
}

// ListPendingTasks 列出待审核任务
func (m *ReviewTaskManager) ListPendingTasks() []ReviewTask {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []ReviewTask
	for _, task := range m.tasks {
		if task.Status == ReviewPending {
			result = append(result, *task)
		}
	}
	return result
}
