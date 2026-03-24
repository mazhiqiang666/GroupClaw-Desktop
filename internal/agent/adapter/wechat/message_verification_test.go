package wechat

import (
	"testing"

	"github.com/mazhiqiang666/GroupClaw-Desktop/internal/agent/windows"
)

// ==================== MessageVerificationRules Basic Tests ====================

func TestMessageVerificationRules_VerifyMessageSend(t *testing.T) {
	pathSystem := NewPathSystem()
	messageClassifier := NewMessageClassifier()
	evidenceCollector := NewEvidenceCollector()
	rules := NewMessageVerificationRules(pathSystem, messageClassifier, evidenceCollector)

	// Before nodes
	beforeNodes := []windows.AccessibleNode{
		{
			Name:     "Old message",
			Role:     "text",
			Bounds:   [4]int{200, 100, 200, 30},
			Children: []windows.AccessibleNode{},
		},
	}

	// After nodes - with new message
	afterNodes := []windows.AccessibleNode{
		{
			Name:     "Old message",
			Role:     "text",
			Bounds:   [4]int{200, 100, 200, 30},
			Children: []windows.AccessibleNode{},
		},
		{
			Name:     "Hello World",
			Role:     "text",
			Bounds:   [4]int{200, 130, 200, 30},
			Children: []windows.AccessibleNode{},
		},
	}

	beforeScreenshot := []byte{1, 2, 3, 4, 5}
	afterScreenshot := []byte{1, 2, 3, 4, 6} // One byte different
	chatAreaBounds := [4]int{200, 100, 200, 30}
	content := "Hello World"

	evidence := rules.VerifyMessageSend(
		beforeNodes, afterNodes,
		beforeScreenshot, afterScreenshot,
		chatAreaBounds, content,
	)

	if evidence.NewMessageNodes != 1 {
		t.Errorf("Expected 1 new message node, got %d", evidence.NewMessageNodes)
	}
	if !evidence.MessageNodeAdded {
		t.Error("Expected MessageNodeAdded to be true")
	}
	if !evidence.MessageContentMatch {
		t.Error("Expected MessageContentMatch to be true")
	}
	if !evidence.ScreenshotChanged {
		t.Error("Expected ScreenshotChanged to be true")
	}
	if evidence.Confidence <= 0 {
		t.Errorf("Expected positive confidence, got %f", evidence.Confidence)
	}
}

// ==================== MessageVerificationRules Dirty Data Tests ====================

func TestMessageVerificationRules_DirtyData_EmptyAndInvalidInputs(t *testing.T) {
	pathSystem := NewPathSystem()
	messageClassifier := NewMessageClassifier()
	evidenceCollector := NewEvidenceCollector()
	rules := NewMessageVerificationRules(pathSystem, messageClassifier, evidenceCollector)

	tests := []struct {
		name             string
		beforeNodes      []windows.AccessibleNode
		afterNodes       []windows.AccessibleNode
		beforeScreenshot []byte
		afterScreenshot  []byte
		chatAreaBounds   [4]int
		content          string
		expectNewNodes   int
	}{
		{
			name:             "Empty before and after nodes",
			beforeNodes:      []windows.AccessibleNode{},
			afterNodes:       []windows.AccessibleNode{},
			beforeScreenshot: []byte{},
			afterScreenshot:  []byte{},
			chatAreaBounds:   [4]int{},
			content:          "test",
			expectNewNodes:   0,
		},
		{
			name: "Empty content",
			beforeNodes: []windows.AccessibleNode{
				{Name: "Old", Role: "text", Bounds: [4]int{200, 100, 200, 30}},
			},
			afterNodes: []windows.AccessibleNode{
				{Name: "Old", Role: "text", Bounds: [4]int{200, 100, 200, 30}},
				{Name: "New", Role: "text", Bounds: [4]int{200, 130, 200, 30}},
			},
			beforeScreenshot: []byte{1, 2, 3},
			afterScreenshot:  []byte{1, 2, 4},
			chatAreaBounds:   [4]int{200, 100, 200, 30},
			content:          "",
			expectNewNodes:   1,
		},
		{
			name:             "Node with empty name",
			beforeNodes:      []windows.AccessibleNode{},
			afterNodes:       []windows.AccessibleNode{{Name: "", Role: "text", Bounds: [4]int{200, 100, 200, 30}}},
			beforeScreenshot: []byte{},
			afterScreenshot:  []byte{},
			chatAreaBounds:   [4]int{},
			content:          "test",
			expectNewNodes:   0, // Empty names are filtered out by FilterMessageAreaNodes
		},
		{
			name: "Same screenshots",
			beforeNodes: []windows.AccessibleNode{
				{Name: "Old", Role: "text", Bounds: [4]int{200, 100, 200, 30}},
			},
			afterNodes: []windows.AccessibleNode{
				{Name: "Old", Role: "text", Bounds: [4]int{200, 100, 200, 30}},
				{Name: "New", Role: "text", Bounds: [4]int{200, 130, 200, 30}},
			},
			beforeScreenshot: []byte{1, 2, 3, 4, 5},
			afterScreenshot:  []byte{1, 2, 3, 4, 5},
			chatAreaBounds:   [4]int{200, 100, 200, 30},
			content:          "test",
			expectNewNodes:   1,
		},
		{
			name:             "Different screenshot lengths",
			beforeNodes:      []windows.AccessibleNode{},
			afterNodes:       []windows.AccessibleNode{{Name: "New", Role: "text", Bounds: [4]int{200, 130, 200, 30}}},
			beforeScreenshot: []byte{1, 2, 3},
			afterScreenshot:  []byte{1, 2, 3, 4, 5, 6},
			chatAreaBounds:   [4]int{},
			content:          "test",
			expectNewNodes:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evidence := rules.VerifyMessageSend(
				tt.beforeNodes, tt.afterNodes,
				tt.beforeScreenshot, tt.afterScreenshot,
				tt.chatAreaBounds, tt.content,
			)

			if evidence.NewMessageNodes != tt.expectNewNodes {
				t.Errorf("Expected %d new message nodes, got %d", tt.expectNewNodes, evidence.NewMessageNodes)
			}
		})
	}
}

func TestMessageVerificationRules_DirtyData_ContentMatching(t *testing.T) {
	pathSystem := NewPathSystem()
	messageClassifier := NewMessageClassifier()
	evidenceCollector := NewEvidenceCollector()
	rules := NewMessageVerificationRules(pathSystem, messageClassifier, evidenceCollector)

	tests := []struct {
		name        string
		afterNodes  []windows.AccessibleNode
		content     string
		expectMatch bool
	}{
		{
			name: "Exact match",
			afterNodes: []windows.AccessibleNode{
				{Name: "Hello World", Role: "text", Bounds: [4]int{200, 130, 200, 30}},
			},
			content:     "Hello World",
			expectMatch: true,
		},
		{
			name: "Partial match",
			afterNodes: []windows.AccessibleNode{
				{Name: "Hello World", Role: "text", Bounds: [4]int{200, 130, 200, 30}},
			},
			content:     "Hello",
			expectMatch: true,
		},
		{
			name: "No match",
			afterNodes: []windows.AccessibleNode{
				{Name: "Different message", Role: "text", Bounds: [4]int{200, 130, 200, 30}},
			},
			content:     "Hello World",
			expectMatch: false,
		},
		{
			name: "Empty node name",
			afterNodes: []windows.AccessibleNode{
				{Name: "", Role: "text", Bounds: [4]int{200, 130, 200, 30}},
			},
			content:     "Hello World",
			expectMatch: false,
		},
		{
			name: "Unicode content",
			afterNodes: []windows.AccessibleNode{
				{Name: "你好世界", Role: "text", Bounds: [4]int{200, 130, 200, 30}},
			},
			content:     "你好",
			expectMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evidence := rules.VerifyMessageSend(
				[]windows.AccessibleNode{}, // before nodes
				tt.afterNodes,
				[]byte{}, // before screenshot
				[]byte{}, // after screenshot
				[4]int{},
				tt.content,
			)

			if evidence.MessageContentMatch != tt.expectMatch {
				t.Errorf("Expected MessageContentMatch=%v, got %v", tt.expectMatch, evidence.MessageContentMatch)
			}
		})
	}
}

// ==================== MessageVerificationRules Complex Scenarios ====================

func TestMessageVerificationRules_ComplexScenarios(t *testing.T) {
	pathSystem := NewPathSystem()
	messageClassifier := NewMessageClassifier()
	evidenceCollector := NewEvidenceCollector()
	rules := NewMessageVerificationRules(pathSystem, messageClassifier, evidenceCollector)

	// Realistic chat scenario
	beforeNodes := []windows.AccessibleNode{
		{Name: "Old message 1", Role: "text", Bounds: [4]int{200, 100, 200, 30}},
		{Name: "Old message 2", Role: "text", Bounds: [4]int{200, 140, 200, 30}},
	}

	afterNodes := []windows.AccessibleNode{
		{Name: "Old message 1", Role: "text", Bounds: [4]int{200, 100, 200, 30}},
		{Name: "Old message 2", Role: "text", Bounds: [4]int{200, 140, 200, 30}},
		{Name: "New message", Role: "text", Bounds: [4]int{200, 180, 200, 30}},
	}

	beforeScreenshot := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	afterScreenshot := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 11} // One byte different
	chatAreaBounds := [4]int{200, 100, 200, 100}
	content := "New message"

	evidence := rules.VerifyMessageSend(
		beforeNodes, afterNodes,
		beforeScreenshot, afterScreenshot,
		chatAreaBounds, content,
	)

	// Verify all evidence points
	if evidence.NewMessageNodes != 1 {
		t.Errorf("Expected 1 new message node, got %d", evidence.NewMessageNodes)
	}
	if !evidence.MessageNodeAdded {
		t.Error("Expected MessageNodeAdded to be true")
	}
	if !evidence.MessageContentMatch {
		t.Error("Expected MessageContentMatch to be true")
	}
	if !evidence.ScreenshotChanged {
		t.Error("Expected ScreenshotChanged to be true")
	}
	if evidence.Confidence < 0.5 {
		t.Errorf("Expected confidence >= 0.5, got %f", evidence.Confidence)
	}
}
