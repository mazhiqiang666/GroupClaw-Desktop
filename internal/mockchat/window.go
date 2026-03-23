package mockchat

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// AccessibilityNode represents a UIA accessibility node
type AccessibilityNode struct {
	ID          string
	Name        string
	Role        string
	Value       string
	Bounds      *Bounds
	Children    []*AccessibilityNode
	Properties  map[string]interface{}
}

// MockChatApp UIA accessibility methods
func (app *MockChatApp) GetAccessibilityNodes() []*AccessibilityNode {
	app.mu.RLock()
	defer app.mu.RUnlock()

	nodes := make([]*AccessibilityNode, 0)

	// Create main window node
	mainWindow := &AccessibilityNode{
		ID:   "main_window",
		Name: "Mock Chat App",
		Role: "Window",
		Bounds: &Bounds{
			X:      0,
			Y:      0,
			Width:  800,
			Height: 600,
		},
		Properties: map[string]interface{}{
			"handle": app.windowHandle,
		},
	}

	// Create conversation list node
	convList := &AccessibilityNode{
		ID:   "conversation_list",
		Name: "会话列表",
		Role: "List",
		Bounds: &Bounds{
			X:      0,
			Y:      0,
			Width:  200,
			Height: 600,
		},
	}

	// Add conversation items
	for _, conv := range app.conversations {
		convNode := &AccessibilityNode{
			ID:   conv.ID,
			Name: conv.DisplayName,
			Role: "ListItem",
			Properties: map[string]interface{}{
				"unread_count": conv.UnreadCount,
				"is_active":    conv.IsActive,
			},
		}
		convList.Children = append(convList.Children, convNode)
	}

	// Create message area node
	msgArea := &AccessibilityNode{
		ID:   "message_area",
		Name: "消息区域",
		Role: "Pane",
		Bounds: &Bounds{
			X:      200,
			Y:      0,
			Width:  600,
			Height: 500,
		},
	}

	// Add messages for active conversation
	if activeConv, exists := app.conversations[app.activeConvID]; exists {
		for i, msg := range activeConv.Messages {
			msgNode := &AccessibilityNode{
				ID:   fmt.Sprintf("msg_%d", i),
				Name: msg.Content,
				Role: "Text",
				Properties: map[string]interface{}{
					"sender_side": msg.SenderSide,
					"timestamp":   msg.Timestamp,
				},
			}
			msgArea.Children = append(msgArea.Children, msgNode)
		}
	}

	// Create input area node
	inputArea := &AccessibilityNode{
		ID:   "input_area",
		Name: "输入区域",
		Role: "Edit",
		Bounds: &Bounds{
			X:      200,
			Y:      500,
			Width:  500,
			Height: 100,
		},
	}

	// Create send button node
	sendButton := &AccessibilityNode{
		ID:   "send_button",
		Name: "发送",
		Role: "Button",
		Bounds: &Bounds{
			X:      700,
			Y:      500,
			Width:  100,
			Height: 100,
		},
	}

	// Assemble the tree
	mainWindow.Children = []*AccessibilityNode{convList, msgArea, inputArea, sendButton}
	nodes = append(nodes, mainWindow)

	return nodes
}

// FindNodeByID finds a node by ID in the accessibility tree
func (app *MockChatApp) FindNodeByID(id string) *AccessibilityNode {
	nodes := app.GetAccessibilityNodes()
	return findNodeByIDRecursive(nodes, id)
}

func findNodeByIDRecursive(nodes []*AccessibilityNode, id string) *AccessibilityNode {
	for _, node := range nodes {
		if node.ID == id {
			return node
		}
		if len(node.Children) > 0 {
			if found := findNodeByIDRecursive(node.Children, id); found != nil {
				return found
			}
		}
	}
	return nil
}

// generateWindowHandle 生成窗口句柄
func generateWindowHandle() uintptr {
	// 生成随机句柄值（模拟）
	b := make([]byte, 8)
	rand.Read(b)
	return uintptr(binaryToUint64(b))
}

// generateMessageID 生成消息 ID
func generateMessageID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("msg_%s", hex.EncodeToString(b))
}

// binaryToUint64 将字节数组转换为 uint64
func binaryToUint64(b []byte) uint64 {
	if len(b) < 8 {
		return 0
	}
	return uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
		uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56
}
