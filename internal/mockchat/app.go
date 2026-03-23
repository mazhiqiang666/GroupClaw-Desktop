package mockchat

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// MockChatApp Mock 聊天应用
type MockChatApp struct {
	windowHandle  uintptr
	conversations map[string]*MockConversation
	activeConvID  string
	uiaMode       UIAMode          // UIA 模式
	dpi           int
	zoom          float64
	mu            sync.RWMutex
}

// UIAMode UIA 模式
type UIAMode string

const (
	UIANormal     UIAMode = "normal"     // 正常模式
	UIAPartialFail UIAMode = "partial_fail" // 部分失效模式
	UIAFullFail   UIAMode = "full_fail"   // 完全失效模式
)

// MockConversation Mock 会话
type MockConversation struct {
	ID          string        `json:"id"`
	DisplayName string        `json:"display_name"`
	UnreadCount int           `json:"unread_count"`
	IsActive    bool          `json:"is_active"`
	Messages    []MockMessage `json:"messages"`
}

// MockMessage Mock 消息
type MockMessage struct {
	ID         string    `json:"id"`
	ConvID     string    `json:"conv_id"`
	SenderSide string    `json:"sender_side"` // customer | agent
	Content    string    `json:"content"`
	Timestamp  time.Time `json:"timestamp"`
	Bounds     *Bounds   `json:"bounds"`
}

// Bounds 窗口坐标
type Bounds struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// NewMockChatApp 创建 Mock 应用
func NewMockChatApp() *MockChatApp {
	app := &MockChatApp{
		windowHandle:  generateWindowHandle(),
		conversations: make(map[string]*MockConversation),
		uiaMode:       UIANormal,
		dpi:           96,
		zoom:          1.0,
	}

	// 初始化默认会话
	app.initDefaultConversations()

	return app
}

// initDefaultConversations 初始化默认会话
func (app *MockChatApp) initDefaultConversations() {
	app.mu.Lock()
	defer app.mu.Unlock()

	// 创建几个测试会话
	convs := []struct {
		id          string
		displayName string
	}{
		{"conv_001", "张三"},
		{"conv_002", "李四"},
		{"conv_003", "王五"},
	}

	for i, c := range convs {
		app.conversations[c.id] = &MockConversation{
			ID:          c.id,
			DisplayName: c.displayName,
			UnreadCount: 0,
			IsActive:    false,
			Messages:    []MockMessage{},
		}
		_ = i // list position
	}

	// 设置第一个会话为激活状态
	if len(convs) > 0 {
		app.activeConvID = convs[0].id
		app.conversations[convs[0].id].IsActive = true
	}
}

// StartHTTPServer 启动 HTTP 注入服务器
func (app *MockChatApp) StartHTTPServer(addr string) error {
	mux := http.NewServeMux()

	// 注入消息接口
	mux.HandleFunc("/api/inject-message", app.handleInjectMessage)
	// 查询会话接口
	mux.HandleFunc("/api/conversations", app.handleGetConversations)
	// 设置 UIA 模式接口
	mux.HandleFunc("/api/set-uia-mode", app.handleSetUIAMode)
	// 健康检查接口
	mux.HandleFunc("/api/health", app.handleHealth)

	return http.ListenAndServe(addr, mux)
}

// handleInjectMessage 注入消息处理器
func (app *MockChatApp) handleInjectMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ConversationID string `json:"conversation_id"`
		Content        string `json:"content"`
		SenderSide     string `json:"sender_side"` // customer | agent
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	app.mu.Lock()
	defer app.mu.Unlock()

	conv, exists := app.conversations[req.ConversationID]
	if !exists {
		http.Error(w, "Conversation not found", http.StatusNotFound)
		return
	}

	// 创建新消息
	msg := MockMessage{
		ID:         generateMessageID(),
		ConvID:     req.ConversationID,
		SenderSide: req.SenderSide,
		Content:    req.Content,
		Timestamp:  time.Now(),
	}

	conv.Messages = append(conv.Messages, msg)

	// 如果是客户消息，增加未读计数
	if req.SenderSide == "customer" && req.ConversationID != app.activeConvID {
		conv.UnreadCount++
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "message_id": msg.ID})
}

// handleGetConversations 查询会话处理器
func (app *MockChatApp) handleGetConversations(w http.ResponseWriter, r *http.Request) {
	app.mu.RLock()
	defer app.mu.RUnlock()

	convs := make([]*MockConversation, 0, len(app.conversations))
	for _, conv := range app.conversations {
		convs = append(convs, conv)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(convs)
}

// handleSetUIAMode 设置 UIA 模式处理器
func (app *MockChatApp) handleSetUIAMode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Mode UIAMode `json:"mode"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	app.mu.Lock()
	defer app.mu.Unlock()
	app.uiaMode = req.Mode

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "uia_mode": string(req.Mode)})
}

// handleHealth 健康检查处理器
func (app *MockChatApp) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// RunGUI 运行 GUI
func (app *MockChatApp) RunGUI() error {
	// 创建并初始化 UI
	ui := NewMockChatUI(app)
	ui.Initialize()

	// 在后台启动 HTTP 服务器
	go func() {
		if err := app.StartHTTPServer(":8081"); err != nil {
			println("HTTP 服务器启动失败:", err.Error())
		}
	}()

	// 运行 GUI
	ui.Run()

	return nil
}
