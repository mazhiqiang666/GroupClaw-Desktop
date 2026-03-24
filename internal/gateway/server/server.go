package server

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mazhiqiang666/GroupClaw-Desktop/internal/gateway/dispatcher"
	"github.com/mazhiqiang666/GroupClaw-Desktop/internal/gateway/handler"
	"github.com/mazhiqiang666/GroupClaw-Desktop/internal/gateway/router"
	"github.com/mazhiqiang666/GroupClaw-Desktop/internal/gateway/session"
	"github.com/mazhiqiang666/GroupClaw-Desktop/pkg/protocol"
)

// Server HTTP/WebSocket 服务器
type Server struct {
	addr     string
	registry *session.Registry
	upgrader websocket.Upgrader

	eventHandler    *handler.EventHandler
	commandDispatcher *dispatcher.CommandDispatcher
	taskRouter      *router.TaskRouter

	mu sync.Mutex
}

// NewServer 创建服务器
func NewServer(addr string) *Server {
	registry := session.NewRegistry()

	return &Server{
		addr:     addr,
		registry: registry,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		eventHandler:    handler.NewEventHandler(registry),
		commandDispatcher: dispatcher.NewCommandDispatcher(registry),
		taskRouter:      router.NewTaskRouter(registry),
	}
}

// Start 启动服务器
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// WebSocket 连接端点
	mux.HandleFunc("/ws", s.handleWebSocket)

	// HTTP API 端点
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/send-reply", s.handleSendReply)

	log.Printf("Gateway 服务器启动在 %s", s.addr)
	return http.ListenAndServe(s.addr, mux)
}

// handleWebSocket WebSocket 连接处理器
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket 升级失败: %v", err)
		return
	}
	defer conn.Close()

	// 创建会话
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		sessionID = protocol.GenerateSessionID()
	}

	sessionObj := &session.Session{
		ID:       sessionID,
		DeviceID: r.URL.Query().Get("device_id"),
		TenantID: r.URL.Query().Get("tenant_id"),
		Conn:     &websocketConnection{conn: conn},
		LastSeen: time.Now(),
	}

	s.registry.Register(sessionObj)
	defer s.registry.Unregister(sessionID)

	log.Printf("Agent 已连接: session=%s", sessionID)

	// 读取消息循环
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("读取消息失败: %v", err)
			break
		}

		// 解析消息
		var env protocol.Envelope
		if err := json.Unmarshal(msg, &env); err != nil {
			log.Printf("解析消息失败: %v", err)
			continue
		}

		// 处理事件
		if env.Kind == protocol.KindEvent {
			s.eventHandler.Handle(&env)
		}

		// 更新最后活跃时间
		s.registry.UpdateLastSeen(sessionID)
	}
}

// handleHealth 健康检查处理器
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleSendReply 发送回复处理器
func (s *Server) handleSendReply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SessionID      string `json:"session_id"`
		ConversationID string `json:"conversation_id"`
		ReplyContent   string `json:"reply_content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	payload := protocol.ReplyExecutePayload{
		ConversationID: req.ConversationID,
		ReplyContent:   req.ReplyContent,
	}

	if err := s.taskRouter.RouteReplyExecute(payload, req.SessionID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// websocketConnection WebSocket 连接包装器
type websocketConnection struct {
	conn *websocket.Conn
}

func (c *websocketConnection) Send(env *protocol.Envelope) error {
	data, err := json.Marshal(env)
	if err != nil {
		return err
	}
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

func (c *websocketConnection) Close() error {
	return c.conn.Close()
}

func (c *websocketConnection) IsClosed() bool {
	return false // 简化实现
}
