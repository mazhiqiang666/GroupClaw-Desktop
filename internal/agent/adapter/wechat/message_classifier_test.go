package wechat

import (
	"testing"

	"github.com/mazhiqiang666/GroupClaw-Desktop/internal/agent/windows"
)

// ==================== Message Classifier Tests ====================

func TestMessageClassifier_ClassifyNode(t *testing.T) {
	mc := NewMessageClassifier()

	tests := []struct {
		name     string
		node     windows.AccessibleNode
		expected NodeType
	}{
		{
			name: "Input box - edit role",
			node: windows.AccessibleNode{
				Name:   "Input",
				Role:   "edit",
				Bounds: [4]int{100, 500, 200, 30},
			},
			expected: NodeTypeInputBox,
		},
		{
			name: "Input box - textbox role",
			node: windows.AccessibleNode{
				Name:   "Input",
				Role:   "textbox",
				Bounds: [4]int{100, 500, 200, 30},
			},
			expected: NodeTypeInputBox,
		},
		{
			name: "Title - top position",
			node: windows.AccessibleNode{
				Name:   "Chat Title",
				Role:   "text",
				Bounds: [4]int{50, 10, 300, 25},
			},
			expected: NodeTypeTitle,
		},
		{
			name: "System prompt - alert role",
			node: windows.AccessibleNode{
				Name:   "Network error",
				Role:   "alert",
				Bounds: [4]int{100, 200, 200, 30},
			},
			expected: NodeTypeSystemPrompt,
		},
		{
			name: "System prompt - status role",
			node: windows.AccessibleNode{
				Name:   "Sending...",
				Role:   "status",
				Bounds: [4]int{100, 200, 200, 30},
			},
			expected: NodeTypeSystemPrompt,
		},
		{
			name: "Message bubble - right aligned",
			node: windows.AccessibleNode{
				Name:   "Hello",
				Role:   "text",
				Bounds: [4]int{300, 100, 150, 40},
			},
			expected: NodeTypeMessageBubble,
		},
		{
			name: "Normal text - text role",
			node: windows.AccessibleNode{
				Name:   "Some text",
				Role:   "text",
				Bounds: [4]int{50, 200, 200, 30},
			},
			expected: NodeTypeNormalText,
		},
		{
			name: "Normal text - static role",
			node: windows.AccessibleNode{
				Name:   "Static text",
				Role:   "static",
				Bounds: [4]int{50, 200, 200, 30},
			},
			expected: NodeTypeNormalText,
		},
		{
			name: "Unknown - invalid bounds",
			node: windows.AccessibleNode{
				Name:   "Unknown",
				Role:   "text",
				Bounds: [4]int{100, 200, 0, 30},
			},
			expected: NodeTypeUnknown,
		},
		{
			name: "Unknown - no matching role",
			node: windows.AccessibleNode{
				Name:   "Unknown",
				Role:   "button",
				Bounds: [4]int{100, 200, 100, 30},
			},
			expected: NodeTypeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mc.ClassifyNode(tt.node)
			if result != tt.expected {
				t.Errorf("ClassifyNode() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMessageClassifier_IsMessageCandidate(t *testing.T) {
	mc := NewMessageClassifier()

	tests := []struct {
		name     string
		node     windows.AccessibleNode
		expected bool
	}{
		{
			name: "Message bubble is candidate",
			node: windows.AccessibleNode{
				Name:   "Hello",
				Role:   "text",
				Bounds: [4]int{300, 100, 150, 40},
			},
			expected: true,
		},
		{
			name: "Normal text is candidate",
			node: windows.AccessibleNode{
				Name:   "Some text",
				Role:   "text",
				Bounds: [4]int{50, 200, 200, 30},
			},
			expected: true,
		},
		{
			name: "Input box is not candidate",
			node: windows.AccessibleNode{
				Name:   "Input",
				Role:   "edit",
				Bounds: [4]int{100, 500, 200, 30},
			},
			expected: false,
		},
		{
			name: "Title is not candidate",
			node: windows.AccessibleNode{
				Name:   "Chat Title",
				Role:   "text",
				Bounds: [4]int{50, 10, 300, 25},
			},
			expected: false,
		},
		{
			name: "System prompt is not candidate",
			node: windows.AccessibleNode{
				Name:   "Network error",
				Role:   "alert",
				Bounds: [4]int{100, 200, 200, 30},
			},
			expected: false,
		},
		{
			name: "Unknown type is not candidate",
			node: windows.AccessibleNode{
				Name:   "Unknown",
				Role:   "button",
				Bounds: [4]int{100, 200, 100, 30},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mc.IsMessageCandidate(tt.node)
			if result != tt.expected {
				t.Errorf("IsMessageCandidate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMessageClassifier_FilterMessageAreaNodes(t *testing.T) {
	mc := NewMessageClassifier()

	tests := []struct {
		name      string
		nodes     []windows.AccessibleNode
		expectLen int
	}{
		{
			name: "Filter out input boxes and titles",
			nodes: []windows.AccessibleNode{
				{Name: "Chat Title", Role: "text", Bounds: [4]int{50, 10, 300, 25}},   // Title - filtered
				{Name: "Input", Role: "edit", Bounds: [4]int{100, 500, 200, 30}},       // Input - filtered
				{Name: "Hello", Role: "text", Bounds: [4]int{300, 100, 150, 40}},       // Message - kept
				{Name: "World", Role: "text", Bounds: [4]int{300, 150, 150, 40}},       // Message - kept
			},
			expectLen: 2,
		},
		{
			name: "Filter out empty name nodes",
			nodes: []windows.AccessibleNode{
				{Name: "", Role: "text", Bounds: [4]int{300, 100, 150, 40}},           // Empty name - filtered
				{Name: "Valid", Role: "text", Bounds: [4]int{300, 150, 150, 40}},       // Valid - kept
			},
			expectLen: 1,
		},
		{
			name: "Filter out invalid bounds",
			nodes: []windows.AccessibleNode{
				{Name: "Invalid", Role: "text", Bounds: [4]int{300, 100, 0, 40}},      // Zero width - filtered
				{Name: "Valid", Role: "text", Bounds: [4]int{300, 150, 150, 40}},       // Valid - kept
			},
			expectLen: 1,
		},
		{
			name: "Keep all valid message nodes",
			nodes: []windows.AccessibleNode{
				{Name: "Hello", Role: "text", Bounds: [4]int{300, 100, 150, 40}},       // Message bubble - kept
				{Name: "World", Role: "static", Bounds: [4]int{50, 200, 200, 30}},      // Normal text - kept
			},
			expectLen: 2,
		},
		{
			name:      "Empty nodes list",
			nodes:     []windows.AccessibleNode{},
			expectLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mc.FilterMessageAreaNodes(tt.nodes, 0)
			if len(result) != tt.expectLen {
				t.Errorf("FilterMessageAreaNodes() returned %d nodes, want %d", len(result), tt.expectLen)
			}
		})
	}
}

// ==================== Message Classifier Dirty Data Tests ====================

func TestMessageClassifier_ClassifyNode_DirtyData(t *testing.T) {
	mc := NewMessageClassifier()

	tests := []struct {
		name     string
		node     windows.AccessibleNode
		expected NodeType
	}{
		{
			name: "Empty name",
			node: windows.AccessibleNode{
				Name:   "",
				Role:   "text",
				Bounds: [4]int{100, 200, 200, 30},
			},
			expected: NodeTypeNormalText,
		},
		{
			name: "Unicode name",
			node: windows.AccessibleNode{
				Name:   "测试消息",
				Role:   "text",
				Bounds: [4]int{300, 100, 150, 40},
			},
			expected: NodeTypeMessageBubble,
		},
		{
			name: "Very long name",
			node: windows.AccessibleNode{
				Name:   "这是一个非常长的消息内容用于测试边界情况",
				Role:   "text",
				Bounds: [4]int{300, 100, 150, 40},
			},
			expected: NodeTypeMessageBubble,
		},
		{
			name: "Special characters in name",
			node: windows.AccessibleNode{
				Name:   "Hello @#$%^&*()",
				Role:   "text",
				Bounds: [4]int{300, 100, 150, 40},
			},
			expected: NodeTypeMessageBubble,
		},
		{
			name: "Case insensitive role match",
			node: windows.AccessibleNode{
				Name:   "Input",
				Role:   "EDIT",
				Bounds: [4]int{100, 500, 200, 30},
			},
			expected: NodeTypeInputBox,
		},
		{
			name: "Negative bounds",
			node: windows.AccessibleNode{
				Name:   "Test",
				Role:   "text",
				Bounds: [4]int{-10, -10, 200, 30},
			},
			expected: NodeTypeNormalText,
		},
		{
			name: "Large bounds",
			node: windows.AccessibleNode{
				Name:   "Test",
				Role:   "text",
				Bounds: [4]int{10000, 10000, 5000, 1000},
			},
			expected: NodeTypeNormalText,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			result := mc.ClassifyNode(tt.node)
			if result != tt.expected {
				t.Errorf("ClassifyNode() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMessageClassifier_FilterMessageAreaNodes_DirtyData(t *testing.T) {
	mc := NewMessageClassifier()

	tests := []struct {
		name      string
		nodes     []windows.AccessibleNode
		expectLen int
	}{
		{
			name: "Nil bounds array",
			nodes: []windows.AccessibleNode{
				{Name: "Test", Role: "text", Bounds: [4]int{}}, // Empty bounds
			},
			expectLen: 0,
		},
		{
			name: "Mixed valid and invalid",
			nodes: []windows.AccessibleNode{
				{Name: "Valid1", Role: "text", Bounds: [4]int{300, 100, 150, 40}},
				{Name: "", Role: "text", Bounds: [4]int{300, 150, 150, 40}},           // Empty name
				{Name: "Valid2", Role: "static", Bounds: [4]int{50, 200, 200, 30}},
				{Name: "Invalid", Role: "text", Bounds: [4]int{300, 200, 0, 30}},      // Zero width
			},
			expectLen: 2,
		},
		{
			name: "All filtered out",
			nodes: []windows.AccessibleNode{
				{Name: "Title", Role: "text", Bounds: [4]int{50, 10, 300, 25}},        // Title
				{Name: "Input", Role: "edit", Bounds: [4]int{100, 500, 200, 30}},      // Input
				{Name: "", Role: "text", Bounds: [4]int{300, 100, 150, 40}},           // Empty name
			},
			expectLen: 0,
		},
		{
			name: "Deeply nested structure (flat nodes)",
			nodes: []windows.AccessibleNode{
				{Name: "Msg1", Role: "text", Bounds: [4]int{300, 100, 150, 40}},
				{Name: "Msg2", Role: "text", Bounds: [4]int{300, 150, 150, 40}},
				{Name: "Msg3", Role: "text", Bounds: [4]int{300, 200, 150, 40}},
			},
			expectLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mc.FilterMessageAreaNodes(tt.nodes, 0)
			if len(result) != tt.expectLen {
				t.Errorf("FilterMessageAreaNodes() returned %d nodes, want %d", len(result), tt.expectLen)
			}
		})
	}
}

func TestMessageClassifier_ComplexScenarios(t *testing.T) {
	mc := NewMessageClassifier()

	// Test realistic chat scenario
	chatNodes := []windows.AccessibleNode{
		// Title area
		{Name: "Chat with 张三", Role: "text", Bounds: [4]int{50, 10, 300, 25}},
		// Input area
		{Name: "Type a message...", Role: "edit", Bounds: [4]int{100, 500, 200, 30}},
		// Message history
		{Name: "Hello", Role: "text", Bounds: [4]int{300, 100, 150, 40}},
		{Name: "Hi there!", Role: "text", Bounds: [4]int{300, 150, 150, 40}},
		{Name: "How are you?", Role: "static", Bounds: [4]int{50, 200, 200, 30}},
		// System message
		{Name: "Message delivered", Role: "status", Bounds: [4]int{200, 250, 100, 20}},
	}

	filtered := mc.FilterMessageAreaNodes(chatNodes, 0)

	// Should keep message nodes and system prompts (Hello, Hi there!, How are you?, Message delivered)
	// Filter removes title (y < 50) and input boxes (role=edit)
	if len(filtered) != 4 {
		t.Errorf("Expected 4 nodes, got %d", len(filtered))
	}

	// Verify the filtered nodes are the correct ones
	expectedNames := []string{"Hello", "Hi there!", "How are you?", "Message delivered"}
	for i, node := range filtered {
		if node.Name != expectedNames[i] {
			t.Errorf("Node %d: expected name '%s', got '%s'", i, expectedNames[i], node.Name)
		}
	}
}

// ==================== Complex Scenario Tests ====================

func TestMessageClassifier_SameNameDifferentTypes(t *testing.T) {
	mc := NewMessageClassifier()

	// Test nodes with same name but different types
	tests := []struct {
		name     string
		node     windows.AccessibleNode
		expected NodeType
	}{
		{
			name: "Same name '张三' as title",
			node: windows.AccessibleNode{
				Name:   "张三",
				Role:   "text",
				Bounds: [4]int{50, 10, 300, 25}, // Top position = title
			},
			expected: NodeTypeTitle,
		},
		{
			name: "Same name '张三' as message bubble",
			node: windows.AccessibleNode{
				Name:   "张三",
				Role:   "text",
				Bounds: [4]int{300, 100, 150, 40}, // Right position = message bubble
			},
			expected: NodeTypeMessageBubble,
		},
		{
			name: "Same name '张三' as normal text",
			node: windows.AccessibleNode{
				Name:   "张三",
				Role:   "static",
				Bounds: [4]int{50, 200, 200, 30}, // Left position = normal text
			},
			expected: NodeTypeNormalText,
		},
		{
			name: "Same name '张三' as input box",
			node: windows.AccessibleNode{
				Name:   "张三",
				Role:   "edit",
				Bounds: [4]int{100, 500, 200, 30}, // Bottom position = input box
			},
			expected: NodeTypeInputBox,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mc.ClassifyNode(tt.node)
			if result != tt.expected {
				t.Errorf("ClassifyNode() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMessageClassifier_DeeplyNestedChatScenario(t *testing.T) {
	mc := NewMessageClassifier()

	// Simulate a complex chat with multiple message types
	chatNodes := []windows.AccessibleNode{
		// Title
		{Name: "Group Chat", Role: "text", Bounds: [4]int{50, 10, 300, 25}},
		// Input area
		{Name: "Type message...", Role: "edit", Bounds: [4]int{100, 550, 200, 30}},
		// Multiple message bubbles (right-aligned)
		{Name: "Hello everyone!", Role: "text", Bounds: [4]int{300, 100, 150, 40}},
		{Name: "How are you?", Role: "text", Bounds: [4]int{300, 150, 150, 40}},
		{Name: "Good morning!", Role: "text", Bounds: [4]int{300, 200, 150, 40}},
		// Left-aligned messages
		{Name: "Hi!", Role: "static", Bounds: [4]int{50, 250, 100, 30}},
		{Name: "Nice to meet you", Role: "static", Bounds: [4]int{50, 290, 150, 30}},
		// System messages
		{Name: "User joined", Role: "status", Bounds: [4]int{200, 330, 100, 20}},
		{Name: "Message sent", Role: "alert", Bounds: [4]int{200, 360, 100, 20}},
		// More messages
		{Name: "See you later", Role: "text", Bounds: [4]int{300, 400, 150, 40}},
	}

	filtered := mc.FilterMessageAreaNodes(chatNodes, 0)

	// Should keep: 4 message bubbles, 2 static texts, 2 system prompts = 8 nodes
	// Filter out: title (y=10), input (edit role)
	expectedCount := 8
	if len(filtered) != expectedCount {
		t.Errorf("Expected %d filtered nodes, got %d", expectedCount, len(filtered))
	}

	// Verify no titles or input boxes in filtered results
	for _, node := range filtered {
		if node.Bounds[1] < 50 {
			t.Errorf("Title node should be filtered out: %s", node.Name)
		}
		if node.Role == "edit" {
			t.Errorf("Input box should be filtered out: %s", node.Name)
		}
	}
}

func TestMessageClassifier_BoundsDriftScenarios(t *testing.T) {
	mc := NewMessageClassifier()

	// Test nodes with slightly drifted bounds
	tests := []struct {
		name     string
		node     windows.AccessibleNode
		expected NodeType
	}{
		{
			name: "Message bubble with small drift",
			node: windows.AccessibleNode{
				Name:   "Hello",
				Role:   "text",
				Bounds: [4]int{302, 102, 148, 38}, // Slightly shifted from original
			},
			expected: NodeTypeMessageBubble,
		},
		{
			name: "Title with small drift",
			node: windows.AccessibleNode{
				Name:   "Chat Title",
				Role:   "text",
				Bounds: [4]int{52, 12, 298, 23}, // Slightly shifted
			},
			expected: NodeTypeTitle,
		},
		{
			name: "Input box with small drift",
			node: windows.AccessibleNode{
				Name:   "Input",
				Role:   "edit",
				Bounds: [4]int{102, 502, 198, 28}, // Slightly shifted
			},
			expected: NodeTypeInputBox,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mc.ClassifyNode(tt.node)
			if result != tt.expected {
				t.Errorf("ClassifyNode() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMessageClassifier_RealisticWeChatScenario(t *testing.T) {
	mc := NewMessageClassifier()

	// Realistic WeChat chat window structure
	chatNodes := []windows.AccessibleNode{
		// Top title bar
		{Name: "张三", Role: "text", Bounds: [4]int{50, 10, 300, 25}},
		// Contact info area
		{Name: "Online", Role: "status", Bounds: [4]int{350, 15, 100, 20}},
		// Message area with mixed content
		{Name: "张三", Role: "text", Bounds: [4]int{300, 80, 150, 40}}, // Message bubble
		{Name: "Hello", Role: "text", Bounds: [4]int{300, 130, 150, 40}},
		{Name: "How are you?", Role: "static", Bounds: [4]int{50, 180, 200, 30}},
		{Name: "I'm good thanks", Role: "text", Bounds: [4]int{300, 220, 150, 40}},
		// System message
		{Name: "Message delivered", Role: "status", Bounds: [4]int{200, 270, 100, 20}},
		// Input area
		{Name: "Type a message...", Role: "edit", Bounds: [4]int{100, 500, 200, 30}},
	}

	filtered := mc.FilterMessageAreaNodes(chatNodes, 0)

	// Expected: 5 message nodes (2 bubbles + 2 static + 1 system)
	// Filter out: title (y=10), contact status (y=15), input (edit)
	expectedCount := 5
	if len(filtered) != expectedCount {
		t.Errorf("Expected %d filtered nodes, got %d", expectedCount, len(filtered))
	}

	// Verify specific nodes are kept
	expectedNames := []string{"张三", "Hello", "How are you?", "I'm good thanks", "Message delivered"}
	for i, node := range filtered {
		if node.Name != expectedNames[i] {
			t.Errorf("Node %d: expected name '%s', got '%s'", i, expectedNames[i], node.Name)
		}
	}
}
