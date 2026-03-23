package wechat

import (
	"testing"

	"github.com/yourorg/auto-customer-service/internal/agent/windows"
)

// ==================== Session Candidate Rules Tests ====================

func TestSessionCandidateRules_IsCandidateConversation(t *testing.T) {
	rules := NewSessionCandidateRules()

	tests := []struct {
		name        string
		node        windows.AccessibleNode
		windowWidth int
		expected    bool
	}{
		{
			name: "Valid candidate - list item in left area",
			node: windows.AccessibleNode{
				Name:     "张三",
				Role:     "list item",
				Bounds:   [4]int{50, 100, 150, 40},
				Children: []windows.AccessibleNode{},
			},
			windowWidth: 800,
			expected:    true,
		},
		{
			name: "Invalid role - not list item",
			node: windows.AccessibleNode{
				Name:     "张三",
				Role:     "text",
				Bounds:   [4]int{50, 100, 150, 40},
				Children: []windows.AccessibleNode{},
			},
			windowWidth: 800,
			expected:    false,
		},
		{
			name: "Invalid - empty name",
			node: windows.AccessibleNode{
				Name:     "",
				Role:     "list item",
				Bounds:   [4]int{50, 100, 150, 40},
				Children: []windows.AccessibleNode{},
			},
			windowWidth: 800,
			expected:    false,
		},
		{
			name: "Invalid - wrong position (right side)",
			node: windows.AccessibleNode{
				Name:     "张三",
				Role:     "list item",
				Bounds:   [4]int{500, 100, 150, 40},
				Children: []windows.AccessibleNode{},
			},
			windowWidth: 800,
			expected:    false,
		},
		{
			name: "Invalid - zero width",
			node: windows.AccessibleNode{
				Name:     "张三",
				Role:     "list item",
				Bounds:   [4]int{50, 100, 0, 40},
				Children: []windows.AccessibleNode{},
			},
			windowWidth: 800,
			expected:    false,
		},
		{
			name: "Valid - ListItem role (uppercase)",
			node: windows.AccessibleNode{
				Name:     "李四",
				Role:     "ListItem",
				Bounds:   [4]int{50, 150, 150, 40},
				Children: []windows.AccessibleNode{},
			},
			windowWidth: 800,
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rules.IsCandidateConversation(tt.node, tt.windowWidth)
			if result != tt.expected {
				t.Errorf("IsCandidateConversation() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSessionCandidateRules_FilterCandidateConversations(t *testing.T) {
	rules := NewSessionCandidateRules()

	nodes := []windows.AccessibleNode{
		{Name: "张三", Role: "list item", Bounds: [4]int{50, 100, 150, 40}},
		{Name: "李四", Role: "text", Bounds: [4]int{50, 150, 150, 40}},       // Invalid role
		{Name: "", Role: "list item", Bounds: [4]int{50, 200, 150, 40}},       // Empty name
		{Name: "王五", Role: "list item", Bounds: [4]int{500, 250, 150, 40}},  // Wrong position
		{Name: "赵六", Role: "list item", Bounds: [4]int{50, 300, 150, 40}},   // Valid
	}

	candidates := rules.FilterCandidateConversations(nodes, 800)

	if len(candidates) != 2 {
		t.Errorf("Expected 2 candidates, got %d", len(candidates))
	}

	if candidates[0].Name != "张三" || candidates[1].Name != "赵六" {
		t.Errorf("Unexpected candidates: %v", candidates)
	}
}

// ==================== Utility Function Tests ====================

func TestGetWindowWidthFromNodes(t *testing.T) {
	// Test with valid nodes
	nodes := []windows.AccessibleNode{
		{Bounds: [4]int{100, 200, 300, 100}}, // x=100, width=300, so window width = 400
	}

	width := GetWindowWidthFromNodes(nodes)
	if width != 400 {
		t.Errorf("Expected width 400, got %d", width)
	}

	// Test with empty nodes
	emptyNodes := []windows.AccessibleNode{}
	width = GetWindowWidthFromNodes(emptyNodes)
	if width != 800 { // default
		t.Errorf("Expected default width 800, got %d", width)
	}

	// Test with small inferred width
	smallNodes := []windows.AccessibleNode{
		{Bounds: [4]int{10, 20, 50, 30}}, // x+width = 60 < 400
	}
	width = GetWindowWidthFromNodes(smallNodes)
	if width != 800 { // should use default
		t.Errorf("Expected default width 800 for small inferred width, got %d", width)
	}
}

func TestConvertFocusEvidenceToDiagnostics(t *testing.T) {
	evidence := FocusVerificationEvidence{
		LocateSource:         "tree_path_name",
		NodeStillExists:      true,
		NodeHasActiveState:   true,
		TitleContainsTarget:  false,
		PanelSwitchDetected:  false,
		MessageAreaVisible:   true,
		Confidence:           0.85,
		EvidenceCount:        3,
	}

	diagnostics := ConvertFocusEvidenceToDiagnostics(evidence)

	if diagnostics["locate_source"] != "tree_path_name" {
		t.Errorf("Expected locate_source 'tree_path_name', got %s", diagnostics["locate_source"])
	}
	if diagnostics["node_still_exists"] != "true" {
		t.Errorf("Expected node_still_exists 'true', got %s", diagnostics["node_still_exists"])
	}
	if diagnostics["confidence"] != "0.85" {
		t.Errorf("Expected confidence '0.85', got %s", diagnostics["confidence"])
	}
}

func TestConvertMessageEvidenceToDiagnostics(t *testing.T) {
	evidence := SendVerificationEvidence{
		NewMessageNodes:      1,
		MessageNodeAdded:     true,
		MessageContentMatch:  true,
		ScreenshotChanged:    true,
		ChatAreaDiff:         0.05,
		Confidence:           0.9,
	}

	diagnostics := ConvertMessageEvidenceToDiagnostics(evidence)

	if diagnostics["new_message_nodes"] != "1" {
		t.Errorf("Expected new_message_nodes '1', got %s", diagnostics["new_message_nodes"])
	}
	if diagnostics["message_node_added"] != "true" {
		t.Errorf("Expected message_node_added 'true', got %s", diagnostics["message_node_added"])
	}
	if diagnostics["confidence"] != "0.90" {
		t.Errorf("Expected confidence '0.90', got %s", diagnostics["confidence"])
	}
}

func TestConvertDeliveryAssessmentToDiagnostics(t *testing.T) {
	assessment := DeliveryAssessment{
		State:      "verified",
		Confidence: 0.85,
		Messages:   []string{"Test message 1", "Test message 2"},
	}

	diagnostics := ConvertDeliveryAssessmentToDiagnostics(assessment)

	if diagnostics["delivery_state"] != "verified" {
		t.Errorf("Expected delivery_state 'verified', got %s", diagnostics["delivery_state"])
	}
	if diagnostics["confidence"] != "0.85" {
		t.Errorf("Expected confidence '0.85', got %s", diagnostics["confidence"])
	}
	if diagnostics["messages"] != "Test message 1; Test message 2" {
		t.Errorf("Expected messages 'Test message 1; Test message 2', got %s", diagnostics["messages"])
	}
}
