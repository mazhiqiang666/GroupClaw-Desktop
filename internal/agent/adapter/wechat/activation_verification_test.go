package wechat

import (
	"testing"

	"github.com/mazhiqiang666/GroupClaw-Desktop/internal/agent/windows"
	"github.com/mazhiqiang666/GroupClaw-Desktop/pkg/protocol"
)

// ==================== ActivationVerificationRules Basic Tests ====================

func TestActivationVerificationRules_VerifySessionActivation(t *testing.T) {
	pathSystem := NewPathSystem()
	evidenceCollector := NewEvidenceCollector()
	rules := NewActivationVerificationRules(pathSystem, evidenceCollector)

	// Create test conversation
	conv := protocol.ConversationRef{
		DisplayName: "张三",
	}

	// Create test nodes - target node exists
	currentNodes := []windows.AccessibleNode{
		{
			Name:     "张三",
			Role:     "list item",
			Bounds:   [4]int{50, 100, 150, 40},
			Children: []windows.AccessibleNode{},
		},
		{
			Name:     "消息区域",
			Role:     "text",
			Bounds:   [4]int{300, 200, 200, 50},
			Children: []windows.AccessibleNode{},
		},
	}

	originalNodes := []windows.AccessibleNode{
		{Name: "李四", Role: "list item", Bounds: [4]int{50, 100, 150, 40}},
	}

	evidence := rules.VerifySessionActivation(conv, currentNodes, originalNodes, "test_source")

	if !evidence.NodeStillExists {
		t.Error("Expected NodeStillExists to be true")
	}
	if !evidence.NodeHasActiveState {
		t.Error("Expected NodeHasActiveState to be true (node in left area)")
	}
	if !evidence.MessageAreaVisible {
		t.Error("Expected MessageAreaVisible to be true")
	}
	if evidence.Confidence <= 0 {
		t.Errorf("Expected positive confidence, got %f", evidence.Confidence)
	}
	if evidence.LocateSource != "test_source" {
		t.Errorf("Expected locate source 'test_source', got %s", evidence.LocateSource)
	}
}

func TestActivationVerificationRules_VerifySessionActivation_NoMatch(t *testing.T) {
	pathSystem := NewPathSystem()
	evidenceCollector := NewEvidenceCollector()
	rules := NewActivationVerificationRules(pathSystem, evidenceCollector)

	conv := protocol.ConversationRef{
		DisplayName: "不存在的人",
	}

	currentNodes := []windows.AccessibleNode{
		{
			Name:     "张三",
			Role:     "list item",
			Bounds:   [4]int{50, 100, 150, 40},
			Children: []windows.AccessibleNode{},
		},
	}

	originalNodes := []windows.AccessibleNode{}

	evidence := rules.VerifySessionActivation(conv, currentNodes, originalNodes, "test_source")

	if evidence.NodeStillExists {
		t.Error("Expected NodeStillExists to be false for non-matching name")
	}
	if evidence.Confidence > 0.5 {
		t.Errorf("Expected low confidence for non-matching name, got %f", evidence.Confidence)
	}
}

// ==================== ActivationVerificationRules Dirty Data Tests ====================

func TestActivationVerificationRules_DirtyData_EmptyAndInvalidNodes(t *testing.T) {
	pathSystem := NewPathSystem()
	evidenceCollector := NewEvidenceCollector()
	rules := NewActivationVerificationRules(pathSystem, evidenceCollector)

	conv := protocol.ConversationRef{DisplayName: "张三"}

	tests := []struct {
		name             string
		currentNodes     []windows.AccessibleNode
		originalNodes    []windows.AccessibleNode
		expectConfidence float64
	}{
		{
			name:             "Empty current nodes",
			currentNodes:     []windows.AccessibleNode{},
			originalNodes:    []windows.AccessibleNode{},
			expectConfidence: 0.0,
		},
		{
			name: "Node with empty name",
			currentNodes: []windows.AccessibleNode{
				{Name: "", Role: "list item", Bounds: [4]int{50, 100, 150, 40}},
			},
			originalNodes:    []windows.AccessibleNode{},
			expectConfidence: 0.0,
		},
		{
			name: "Node with wrong role",
			currentNodes: []windows.AccessibleNode{
				{Name: "张三", Role: "text", Bounds: [4]int{50, 100, 150, 40}},
			},
			originalNodes:    []windows.AccessibleNode{},
			expectConfidence: 0.0,
		},
		{
			name: "Node with invalid bounds",
			currentNodes: []windows.AccessibleNode{
				{Name: "张三", Role: "list item", Bounds: [4]int{}},
			},
			originalNodes:    []windows.AccessibleNode{},
			expectConfidence: 0.0,
		},
		{
			name: "Multiple nodes with one match",
			currentNodes: []windows.AccessibleNode{
				{Name: "李四", Role: "list item", Bounds: [4]int{50, 100, 150, 40}},
				{Name: "张三", Role: "list item", Bounds: [4]int{50, 150, 150, 40}},
				{Name: "王五", Role: "list item", Bounds: [4]int{50, 200, 150, 40}},
			},
			originalNodes:    []windows.AccessibleNode{},
			expectConfidence: 0.5, // Only node exists, no other evidence
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evidence := rules.VerifySessionActivation(conv, tt.currentNodes, tt.originalNodes, "test")

			if evidence.Confidence < tt.expectConfidence {
				t.Errorf("Expected confidence >= %f, got %f", tt.expectConfidence, evidence.Confidence)
			}
		})
	}
}

func TestActivationVerificationRules_DirtyData_PanelSwitchDetection(t *testing.T) {
	pathSystem := NewPathSystem()
	evidenceCollector := NewEvidenceCollector()
	rules := NewActivationVerificationRules(pathSystem, evidenceCollector)

	conv := protocol.ConversationRef{DisplayName: "张三"}

	tests := []struct {
		name          string
		originalNodes []windows.AccessibleNode
		currentNodes  []windows.AccessibleNode
		expectSwitch  bool
	}{
		{
			name:          "Empty original nodes",
			originalNodes: []windows.AccessibleNode{},
			currentNodes:  []windows.AccessibleNode{{Name: "张三", Role: "list item", Bounds: [4]int{50, 100, 150, 40}}},
			expectSwitch:  false,
		},
		{
			name:          "Empty current nodes",
			originalNodes: []windows.AccessibleNode{{Name: "张三", Role: "list item", Bounds: [4]int{50, 100, 150, 40}}},
			currentNodes:  []windows.AccessibleNode{},
			expectSwitch:  false, // Empty current nodes means no match
		},
		{
			name: "Significant node count decrease",
			originalNodes: []windows.AccessibleNode{
				{Name: "A", Role: "list item", Bounds: [4]int{50, 100, 150, 40}},
				{Name: "B", Role: "list item", Bounds: [4]int{50, 150, 150, 40}},
				{Name: "C", Role: "list item", Bounds: [4]int{50, 200, 150, 40}},
				{Name: "D", Role: "list item", Bounds: [4]int{50, 250, 150, 40}},
				{Name: "E", Role: "list item", Bounds: [4]int{50, 300, 150, 40}},
			},
			currentNodes: []windows.AccessibleNode{
				{Name: "张三", Role: "list item", Bounds: [4]int{50, 100, 150, 40}},
			},
			expectSwitch: true,
		},
		{
			name: "Significant node count increase",
			originalNodes: []windows.AccessibleNode{
				{Name: "张三", Role: "list item", Bounds: [4]int{50, 100, 150, 40}},
			},
			currentNodes: []windows.AccessibleNode{
				{Name: "A", Role: "list item", Bounds: [4]int{50, 100, 150, 40}},
				{Name: "B", Role: "list item", Bounds: [4]int{50, 150, 150, 40}},
				{Name: "C", Role: "list item", Bounds: [4]int{50, 200, 150, 40}},
				{Name: "D", Role: "list item", Bounds: [4]int{50, 250, 150, 40}},
				{Name: "E", Role: "list item", Bounds: [4]int{50, 300, 150, 40}},
			},
			expectSwitch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evidence := rules.VerifySessionActivation(conv, tt.currentNodes, tt.originalNodes, "test")

			if evidence.PanelSwitchDetected != tt.expectSwitch {
				t.Errorf("Expected PanelSwitchDetected=%v, got %v", tt.expectSwitch, evidence.PanelSwitchDetected)
			}
		})
	}
}

// ==================== ActivationVerificationRules Complex Scenarios ====================

func TestActivationVerificationRules_ComplexScenarios(t *testing.T) {
	pathSystem := NewPathSystem()
	evidenceCollector := NewEvidenceCollector()
	rules := NewActivationVerificationRules(pathSystem, evidenceCollector)

	conv := protocol.ConversationRef{DisplayName: "张三"}

	// Complex scenario: multiple panels with different states
	tests := []struct {
		name          string
		currentNodes  []windows.AccessibleNode
		originalNodes []windows.AccessibleNode
		expectExists  bool
		expectActive  bool
		expectTitle   bool
		expectPanel   bool
	}{
		{
			name: "Full activation with all evidence",
			currentNodes: []windows.AccessibleNode{
				{Name: "张三", Role: "list item", Bounds: [4]int{50, 100, 150, 40}},
				{Name: "张三", Role: "text", Bounds: [4]int{50, 10, 300, 25}}, // Title
				{Name: "消息区域", Role: "text", Bounds: [4]int{300, 200, 200, 50}},
			},
			originalNodes: []windows.AccessibleNode{
				{Name: "李四", Role: "list item", Bounds: [4]int{50, 100, 150, 40}},
				{Name: "王五", Role: "list item", Bounds: [4]int{50, 150, 150, 40}},
			},
			expectExists:  true,
			expectActive:  true,
			expectTitle:   true,
			expectPanel:   true,
		},
		{
			name: "Node exists but not in active position",
			currentNodes: []windows.AccessibleNode{
				{Name: "张三", Role: "list item", Bounds: [4]int{300, 100, 150, 40}},
			},
			originalNodes: []windows.AccessibleNode{},
			expectExists:  true,
			expectActive:  false,
			expectTitle:   false,
			expectPanel:   false,
		},
		{
			name: "Same node count (no panel switch)",
			currentNodes: []windows.AccessibleNode{
				{Name: "张三", Role: "list item", Bounds: [4]int{50, 100, 150, 40}},
			},
			originalNodes: []windows.AccessibleNode{
				{Name: "李四", Role: "list item", Bounds: [4]int{50, 100, 150, 40}},
			},
			expectExists:  true,
			expectActive:  true,
			expectTitle:   false,
			expectPanel:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evidence := rules.VerifySessionActivation(conv, tt.currentNodes, tt.originalNodes, "test")

			if evidence.NodeStillExists != tt.expectExists {
				t.Errorf("NodeStillExists = %v, want %v", evidence.NodeStillExists, tt.expectExists)
			}
			if evidence.NodeHasActiveState != tt.expectActive {
				t.Errorf("NodeHasActiveState = %v, want %v", evidence.NodeHasActiveState, tt.expectActive)
			}
			if evidence.TitleContainsTarget != tt.expectTitle {
				t.Errorf("TitleContainsTarget = %v, want %v", evidence.TitleContainsTarget, tt.expectTitle)
			}
			if evidence.PanelSwitchDetected != tt.expectPanel {
				t.Errorf("PanelSwitchDetected = %v, want %v", evidence.PanelSwitchDetected, tt.expectPanel)
			}
		})
	}
}
