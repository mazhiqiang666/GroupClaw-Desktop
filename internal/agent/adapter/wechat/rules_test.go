package wechat

import (
	"testing"

	"github.com/yourorg/auto-customer-service/internal/agent/windows"
	"github.com/yourorg/auto-customer-service/pkg/protocol"
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

// ==================== Positioning Strategy Rules Tests ====================

func TestPositioningStrategyRules_FindNodeByStrategy(t *testing.T) {
	pathSystem := NewPathSystem()
	rules := NewPositioningStrategyRules(pathSystem)

	// Create test nodes with TreePath
	nodes := []windows.AccessibleNode{
		{
			Name:     "张三",
			Role:     "list item",
			Bounds:   [4]int{50, 100, 150, 40},
			TreePath: "[0]",
		},
		{
			Name:     "李四",
			Role:     "list item",
			Bounds:   [4]int{50, 150, 150, 40},
			TreePath: "[1]",
		},
	}

	// Test strategy 1: Tree path + name match
	conv := protocol.ConversationRef{
		DisplayName:         "张三",
		ListNeighborhoodHint: []string{"[0]", "bounds:50_100_150_40"},
	}

	result := rules.FindNodeByStrategy(nodes, conv)
	if result.Node == nil || result.Node.Name != "张三" {
		t.Errorf("Expected to find '张三' by tree path, got %v", result.Node)
	}
	if result.Source != "tree_path_name" {
		t.Errorf("Expected source 'tree_path_name', got %s", result.Source)
	}

	// Test strategy 2: Bounds match (using mismatched tree path to force bounds match)
	nodesWithoutTreePath := []windows.AccessibleNode{
		{
			Name:     "李四",
			Role:     "list item",
			Bounds:   [4]int{50, 150, 150, 40},
			TreePath: "[99]", // Different from hint to skip tree_path_name strategy
		},
	}
	// Use a tree path hint that doesn't match the node's TreePath
	// so it falls through to bounds_match strategy
	conv2 := protocol.ConversationRef{
		DisplayName:         "李四",
		ListNeighborhoodHint: []string{"[1]", "bounds:50_150_150_40"},
	}

	result2 := rules.FindNodeByStrategy(nodesWithoutTreePath, conv2)
	if result2.Node == nil || result2.Node.Name != "李四" {
		t.Errorf("Expected to find '李四' by bounds match, got %v", result2.Node)
	}
	if result2.Source != "bounds_match" {
		t.Errorf("Expected source 'bounds_match', got %s", result2.Source)
	}

	// Test strategy 4: Name match only
	conv3 := protocol.ConversationRef{
		DisplayName: "张三",
	}

	result3 := rules.FindNodeByStrategy(nodes, conv3)
	if result3.Node == nil || result3.Node.Name != "张三" {
		t.Errorf("Expected to find '张三' by name match, got %v", result3.Node)
	}
	if result3.Source != "name_match" {
		t.Errorf("Expected source 'name_match', got %s", result3.Source)
	}

	// Test no match
	conv4 := protocol.ConversationRef{
		DisplayName: "不存在的人",
	}

	result4 := rules.FindNodeByStrategy(nodes, conv4)
	if result4.Node != nil {
		t.Errorf("Expected no match, got %v", result4.Node)
	}
	if result4.Source != "not_found" {
		t.Errorf("Expected source 'not_found', got %s", result4.Source)
	}
}

func TestPositioningStrategyRules_CalculateClickPosition(t *testing.T) {
	pathSystem := NewPathSystem()
	rules := NewPositioningStrategyRules(pathSystem)

	// Test valid node
	node := &windows.AccessibleNode{
		Name:   "Test",
		Bounds: [4]int{100, 200, 200, 100}, // x=100, y=200, width=200, height=100
	}

	clickX, clickY, ok := rules.CalculateClickPosition(node)
	if !ok {
		t.Error("Expected valid click position")
	}
	if clickX != 200 { // 100 + 200/2 = 200
		t.Errorf("Expected clickX=200, got %d", clickX)
	}
	if clickY != 250 { // 200 + 100/2 = 250
		t.Errorf("Expected clickY=250, got %d", clickY)
	}

	// Test nil node
	_, _, ok = rules.CalculateClickPosition(nil)
	if ok {
		t.Error("Expected invalid for nil node")
	}

	// Test node with zero width/height (invalid for clicking)
	nodeInvalid := &windows.AccessibleNode{
		Name:   "Test",
		Bounds: [4]int{100, 200, 0, 0}, // Zero width and height
	}
	_, _, ok = rules.CalculateClickPosition(nodeInvalid)
	// Zero dimensions are invalid for clicking
	if ok {
		t.Error("Expected invalid for node with zero dimensions")
	}
}

// ==================== Activation Verification Rules Tests ====================

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

// ==================== Message Verification Rules Tests ====================

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

// ==================== Delivery Assessment Rules Tests ====================

func TestDeliveryAssessmentRules_AssessDeliveryState(t *testing.T) {
	rules := NewDeliveryAssessmentRules()

	// Test verified state
	focusEvidence := FocusVerificationEvidence{
		NodeStillExists:    true,
		NodeHasActiveState: true,
		Confidence:         0.9,
		EvidenceCount:      2,
	}
	messageEvidence := SendVerificationEvidence{
		NewMessageNodes:   1,
		MessageNodeAdded:  true,
		ScreenshotChanged: true,
		Confidence:        0.9,
	}

	assessment := rules.AssessDeliveryState(focusEvidence, messageEvidence)

	if assessment.State != "verified" {
		t.Errorf("Expected state 'verified', got '%s'", assessment.State)
	}
	if assessment.Confidence < 0.8 {
		t.Errorf("Expected confidence >= 0.8, got %f", assessment.Confidence)
	}

	// Test sent_unverified state
	focusEvidence.Confidence = 0.5
	messageEvidence.Confidence = 0.5
	assessment = rules.AssessDeliveryState(focusEvidence, messageEvidence)

	if assessment.State != "sent_unverified" {
		t.Errorf("Expected state 'sent_unverified', got '%s'", assessment.State)
	}

	// Test unknown state
	focusEvidence.Confidence = 0.1
	messageEvidence.Confidence = 0.1
	assessment = rules.AssessDeliveryState(focusEvidence, messageEvidence)

	if assessment.State != "unknown" {
		t.Errorf("Expected state 'unknown', got '%s'", assessment.State)
	}
}

func TestDeliveryAssessmentRules_AssessFocusOnlyState(t *testing.T) {
	rules := NewDeliveryAssessmentRules()

	// Test verified state
	focusEvidence := FocusVerificationEvidence{
		NodeStillExists:    true,
		NodeHasActiveState: true,
		Confidence:         0.9,
		EvidenceCount:      2,
	}

	assessment := rules.AssessFocusOnlyState(focusEvidence)

	if assessment.State != "verified" {
		t.Errorf("Expected state 'verified', got '%s'", assessment.State)
	}
	if assessment.Confidence != 0.9 {
		t.Errorf("Expected confidence 0.9, got %f", assessment.Confidence)
	}

	// Test unknown state
	focusEvidence.Confidence = 0.1
	assessment = rules.AssessFocusOnlyState(focusEvidence)

	if assessment.State != "unknown" {
		t.Errorf("Expected state 'unknown', got '%s'", assessment.State)
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
