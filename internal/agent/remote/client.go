package remote

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/mazhiqiang666/GroupClaw-Desktop/internal/agent/monitor"
	"github.com/mazhiqiang666/GroupClaw-Desktop/internal/agent/session"
)

// GatewayClient 网关客户端（实现RemoteAgentClient接口）
type GatewayClient struct {
	endpoint   string
	httpClient *http.Client
	timeout    time.Duration
}

// NewGatewayClient 创建新的网关客户端
func NewGatewayClient(endpoint string, timeout time.Duration) *GatewayClient {
	return &GatewayClient{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

// GetReply 从网关获取回复
func (gc *GatewayClient) GetReply(ctx context.Context, context monitor.AgentContext) (string, error) {
	log.Printf("调用远端agent获取回复: contact=%s", context.ContactName)

	// 构建请求
	request := GatewayRequest{
		TaskID:      fmt.Sprintf("monitor_%d", time.Now().UnixNano()),
		ContactID:   context.ContactID,
		ContactName: context.ContactName,
		Messages:    convertMessages(context.MessageHistory),
		Timestamp:   time.Now(),
	}

	// 序列化请求
	requestBody, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %v", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", gc.endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return "", fmt.Errorf("创建HTTP请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "GroupClaw-Monitor/1.0")

	// 发送请求
	resp, err := gc.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("网关返回错误状态码: %d, 响应: %s", resp.StatusCode, string(responseBody))
	}

	// 解析响应
	var gatewayResp GatewayResponse
	if err := json.Unmarshal(responseBody, &gatewayResp); err != nil {
		return "", fmt.Errorf("解析响应失败: %v", err)
	}

	if !gatewayResp.Success {
		return "", fmt.Errorf("网关返回失败: %s", gatewayResp.Error)
	}

	log.Printf("远端agent回复成功: length=%d", len(gatewayResp.Reply))
	return gatewayResp.Reply, nil
}

// GetEndpoint 获取端点地址
func (gc *GatewayClient) GetEndpoint() string {
	return gc.endpoint
}

// GatewayRequest 网关请求结构
type GatewayRequest struct {
	TaskID      string          `json:"task_id"`
	ContactID   string          `json:"contact_id"`
	ContactName string          `json:"contact_name"`
	Messages    []GatewayMessage `json:"messages"`
	Timestamp   time.Time       `json:"timestamp"`
}

// GatewayMessage 网关消息结构
type GatewayMessage struct {
	ID        string    `json:"id"`
	Sender    string    `json:"sender"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	IsOutgoing bool     `json:"is_outgoing"`
}

// GatewayResponse 网关响应结构
type GatewayResponse struct {
	TaskID  string `json:"task_id"`
	Success bool   `json:"success"`
	Reply   string `json:"reply"`
	Error   string `json:"error,omitempty"`
}

// convertMessages 转换会话消息为网关消息
func convertMessages(sessionMessages []session.Message) []GatewayMessage {
	var gatewayMessages []GatewayMessage
	for _, msg := range sessionMessages {
		gatewayMessages = append(gatewayMessages, GatewayMessage{
			ID:        msg.ID,
			Sender:    msg.Sender,
			Content:   msg.Content,
			Timestamp: msg.Timestamp,
			IsOutgoing: msg.IsOutgoing,
		})
	}
	return gatewayMessages
}

// MockClient 模拟客户端（用于测试）
type MockClient struct {
	endpoint string
	replies  map[string]string // contactID -> 预设回复
}

// NewMockClient 创建模拟客户端
func NewMockClient(endpoint string) *MockClient {
	return &MockClient{
		endpoint: endpoint,
		replies:  make(map[string]string),
	}
}

// SetReply 设置联系人的预设回复
func (mc *MockClient) SetReply(contactID, reply string) {
	mc.replies[contactID] = reply
}

// GetReply 获取模拟回复
func (mc *MockClient) GetReply(ctx context.Context, context monitor.AgentContext) (string, error) {
	log.Printf("[MOCK] 模拟调用远端agent: contact=%s", context.ContactName)

	// 模拟网络延迟
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(100 * time.Millisecond):
		// 继续
	}

	// 检查是否有预设回复
	if reply, exists := mc.replies[context.ContactID]; exists {
		log.Printf("[MOCK] 返回预设回复: length=%d", len(reply))
		return reply, nil
	}

	// 默认回复
	defaultReply := fmt.Sprintf("这是对 %s 消息的自动回复。时间：%s", context.ContactName, time.Now().Format("15:04:05"))
	log.Printf("[MOCK] 返回默认回复: length=%d", len(defaultReply))
	return defaultReply, nil
}

// GetEndpoint 获取端点地址
func (mc *MockClient) GetEndpoint() string {
	return mc.endpoint
}
