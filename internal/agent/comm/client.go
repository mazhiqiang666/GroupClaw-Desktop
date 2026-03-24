package comm

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/mazhiqiang666/GroupClaw-Desktop/pkg/protocol"
)

// WebSocketClient WebSocket 客户端
type WebSocketClient struct {
	conn      *websocket.Conn
	sessionID string
	deviceID  string
	tenantID  string
	mu        sync.Mutex
	handlers  map[protocol.PayloadType]func(*protocol.Envelope)
	closed    bool
}

// NewWebSocketClient 创建 WebSocket 客户端
func NewWebSocketClient(sessionID, deviceID, tenantID string) *WebSocketClient {
	return &WebSocketClient{
		sessionID: sessionID,
		deviceID:  deviceID,
		tenantID:  tenantID,
		handlers:  make(map[protocol.PayloadType]func(*protocol.Envelope)),
	}
}

// Connect 连接到 Gateway
func (c *WebSocketClient) Connect(ctx context.Context, addr string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return errors.New("already connected")
	}

	// 构建 WebSocket URL
	url := "ws://" + addr + "/ws"
	if c.sessionID != "" {
		url += "?session_id=" + c.sessionID
	}
	if c.deviceID != "" {
		url += "&device_id=" + c.deviceID
	}
	if c.tenantID != "" {
		url += "&tenant_id=" + c.tenantID
	}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, url, nil)
	if err != nil {
		return err
	}

	c.conn = conn
	c.closed = false

	// 启动读取消息的 goroutine
	go c.readLoop(ctx)

	log.Printf("WebSocket 客户端已连接: %s", addr)
	return nil
}

// readLoop 读取消息循环
func (c *WebSocketClient) readLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if c.conn == nil || c.closed {
				return
			}

			_, msg, err := c.conn.ReadMessage()
			if err != nil {
				if !c.closed {
					log.Printf("读取消息失败: %v", err)
				}
				return
			}

			// 解析消息
			var env protocol.Envelope
			if err := json.Unmarshal(msg, &env); err != nil {
				log.Printf("解析消息失败: %v", err)
				continue
			}

			// 调用处理器
			c.mu.Lock()
			handler, exists := c.handlers[env.PayloadType]
			c.mu.Unlock()
			if exists {
				handler(&env)
			}
		}
	}
}

// RegisterHandler 注册消息处理器
func (c *WebSocketClient) RegisterHandler(payloadType protocol.PayloadType, handler func(*protocol.Envelope)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[payloadType] = handler
}

// Send 发送消息
func (c *WebSocketClient) Send(env *protocol.Envelope) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil || c.closed {
		return errors.New("not connected")
	}

	data, err := json.Marshal(env)
	if err != nil {
		return err
	}

	return c.conn.WriteMessage(websocket.TextMessage, data)
}

// SendEvent 发送事件
func (c *WebSocketClient) SendEvent(payloadType protocol.PayloadType, payload protocol.Payload) error {
	env, err := protocol.NewEnvelope(protocol.KindEvent, payloadType, payload)
	if err != nil {
		return err
	}
	env.DeviceID = c.deviceID
	env.TenantID = c.tenantID
	return c.Send(env)
}

// Close 关闭连接
func (c *WebSocketClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.closed = true
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// IsConnected 检查是否已连接
func (c *WebSocketClient) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil && !c.closed
}

// DeviceID 获取设备ID
func (c *WebSocketClient) DeviceID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.deviceID
}

// TenantID 获取租户ID
func (c *WebSocketClient) TenantID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tenantID
}

// SessionID 获取会话ID
func (c *WebSocketClient) SessionID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.sessionID
}
