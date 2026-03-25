package adapter

import (
	"time"

	"github.com/mazhiqiang666/GroupClaw-Desktop/pkg/protocol"
)

// ChatAdapter 聊天软件适配器接口
type ChatAdapter interface {
	// 基础信息
	Name() string
	Version() string
	SupportedApps() []string

	// 生命周期
	Init(config Config) Result
	Destroy() Result
	IsAvailable() Result

	// 核心操作
	Detect() ([]protocol.AppInstanceRef, Result)
	Scan(instance protocol.AppInstanceRef) ([]protocol.ConversationRef, Result)
	Focus(conv protocol.ConversationRef) Result
	Read(conv protocol.ConversationRef, limit int) ([]protocol.MessageObs, Result)
	Send(conv protocol.ConversationRef, content string, taskID string) Result
	Verify(conv protocol.ConversationRef, content string, timeout time.Duration) (*protocol.MessageObs, Result)
	CaptureDiagnostics() (map[string]string, Result)
}

// Config 适配器配置
type Config struct {
	EnableNative bool `json:"enable_native"`
	EnableOCR    bool `json:"enable_ocr"`
	EnableVisual bool `json:"enable_visual"`
	PollInterval int  `json:"poll_interval"`
	TimeoutMs    int  `json:"timeout_ms"`
}

// Result 统一返回对象
type Result struct {
	Status       Status            `json:"status"`
	ReasonCode   ReasonCode        `json:"reason_code"`
	Confidence   float64           `json:"confidence"`
	DataSource   DataSource        `json:"data_source"`
	Diagnostics  []Diagnostic      `json:"diagnostics"`
	Artifacts    map[string]string `json:"artifacts"`
	Error        string            `json:"error"`
	ElapsedMs    int64             `json:"elapsed_ms"`
}

// Status 执行状态
type Status string

const (
	StatusSuccess   Status = "success"
	StatusPartial   Status = "partial"
	StatusFailed    Status = "failed"
	StatusTimeout   Status = "timeout"
	StatusSkipped   Status = "skipped"
)

// ReasonCode 原因码
type ReasonCode string

const (
	ReasonOK                  ReasonCode = "OK"
	ReasonAppNotRunning       ReasonCode = "APP_NOT_RUNNING"
	ReasonWindowNotFound      ReasonCode = "WINDOW_NOT_FOUND"
	ReasonConvNotFound        ReasonCode = "CONV_NOT_FOUND"
	ReasonSendFailed          ReasonCode = "SEND_FAILED"
	ReasonVerifyFailed        ReasonCode = "VERIFY_FAILED"
	ReasonInputBoxNotConfident ReasonCode = "INPUT_BOX_NOT_CONFIDENT"
	ReasonInputBoxProbeFailed ReasonCode = "INPUT_BOX_PROBE_FAILED"
	ReasonTextInjectionFailed ReasonCode = "TEXT_INJECTION_FAILED"
	ReasonSendActionFailed    ReasonCode = "SEND_ACTION_FAILED"
	ReasonSendNotVerified     ReasonCode = "SEND_NOT_VERIFIED"
	ReasonSendVerified        ReasonCode = "SEND_VERIFIED"
)

// DataSource 数据来源
type DataSource string

const (
	SourceNative DataSource = "native"
	SourceOCR    DataSource = "ocr"
	SourceVisual DataSource = "visual"
)

// Diagnostic 诊断信息
type Diagnostic struct {
	Timestamp time.Time         `json:"timestamp"`
	Level     string            `json:"level"`
	Message   string            `json:"message"`
	Context   map[string]string `json:"context"`
}
