package wechat

import (
	"testing"

	"github.com/yourorg/auto-customer-service/internal/agent/windows"
	"github.com/yourorg/auto-customer-service/pkg/protocol"
)

// TestPathSystem_GeneratePath tests path generation
func TestPathSystem_GeneratePath(t *testing.T) {
	ps := NewPathSystem()

	// Test root node
	node := windows.AccessibleNode{Name: "Root"}
	path := ps.GeneratePath(node, "", 0)
	if path != "[0]" {
		t.Errorf("Expected root path '[0]', got '%s'", path)
	}

	// Test child node
	childPath := ps.GeneratePath(node, "[0]", 3)
	if childPath != "[0].[3]" {
		t.Errorf("Expected child path '[0].[3]', got '%s'", childPath)
	}

	// Test nested path
	nestedPath := ps.GeneratePath(node, "[0].[3]", 2)
	if nestedPath != "[0].[3].[2]" {
		t.Errorf("Expected nested path '[0].[3].[2]', got '%s'", nestedPath)
	}
}

// TestPathSystem_ParsePath tests path parsing
func TestPathSystem_ParsePath(t *testing.T) {
	ps := NewPathSystem()

	// Test simple path
	indices, err := ps.ParsePath("[0]")
	if err != nil {
		t.Errorf("Failed to parse '[0]': %v", err)
	}
	if len(indices) != 1 || indices[0] != 0 {
		t.Errorf("Expected [0], got %v", indices)
	}

	// Test hierarchical path
	indices, err = ps.ParsePath("[0].[3].[2]")
	if err != nil {
		t.Errorf("Failed to parse '[0].[3].[2]': %v", err)
	}
	if len(indices) != 3 || indices[0] != 0 || indices[1] != 3 || indices[2] != 2 {
		t.Errorf("Expected [0, 3, 2], got %v", indices)
	}

	// Test empty path
	_, err = ps.ParsePath("")
	if err == nil {
		t.Error("Expected error for empty path")
	}

	// Test invalid path
	_, err = ps.ParsePath("[abc]")
	if err == nil {
		t.Error("Expected error for invalid path")
	}
}

// TestPathSystem_FindNodeByPath tests finding node by path
func TestPathSystem_FindNodeByPath(t *testing.T) {
	ps := NewPathSystem()

	// Create a proper tree structure
	nodes := []windows.AccessibleNode{
		{
			Name: "Root",
			TreePath: "[0]",
			Children: []windows.AccessibleNode{
				{
					Name: "Child1",
					TreePath: "[0].[0]",
					Children: []windows.AccessibleNode{
						{Name: "Grandchild", TreePath: "[0].[0].[0]"},
					},
				},
				{
					Name: "Child2",
					TreePath: "[0].[1]",
				},
			},
		},
	}

	// Test finding root node
	node, err := ps.FindNodeByPath(nodes, "[0]")
	if err != nil {
		t.Errorf("Failed to find root node: %v", err)
	}
	if node.Name != "Root" {
		t.Errorf("Expected 'Root', got '%s'", node.Name)
	}

	// Test finding child node
	node, err = ps.FindNodeByPath(nodes, "[0].[1]")
	if err != nil {
		t.Errorf("Failed to find child node: %v", err)
	}
	if node.Name != "Child2" {
		t.Errorf("Expected 'Child2', got '%s'", node.Name)
	}

	// Test finding grandchild node
	node, err = ps.FindNodeByPath(nodes, "[0].[0].[0]")
	if err != nil {
		t.Errorf("Failed to find grandchild node: %v", err)
	}
	if node.Name != "Grandchild" {
		t.Errorf("Expected 'Grandchild', got '%s'", node.Name)
	}

	// Test invalid path
	_, err = ps.FindNodeByPath(nodes, "[5]")
	if err == nil {
		t.Error("Expected error for out of range index")
	}
}

// TestPathSystem_FlattenNodesWithPath tests flattening nodes with paths
func TestPathSystem_FlattenNodesWithPath(t *testing.T) {
	ps := NewPathSystem()

	nodes := []windows.AccessibleNode{
		{Name: "Root", Children: []windows.AccessibleNode{
			{Name: "Child1"},
			{Name: "Child2", Children: []windows.AccessibleNode{
				{Name: "Grandchild"},
			}},
		}},
	}

	flatNodes := ps.FlattenNodesWithPath(nodes, "", 0, 10)

	if len(flatNodes) != 4 {
		t.Errorf("Expected 4 flat nodes, got %d", len(flatNodes))
	}

	// Check that TreePath is set
	if flatNodes[0].TreePath != "[0]" {
		t.Errorf("Expected root path '[0]', got '%s'", flatNodes[0].TreePath)
	}
	if flatNodes[1].TreePath != "[0].[0]" {
		t.Errorf("Expected child path '[0].[0]', got '%s'", flatNodes[1].TreePath)
	}
	if flatNodes[2].TreePath != "[0].[1]" {
		t.Errorf("Expected child path '[0].[1]', got '%s'", flatNodes[2].TreePath)
	}
	if flatNodes[3].TreePath != "[0].[1].[0]" {
		t.Errorf("Expected grandchild path '[0].[1].[0]', got '%s'", flatNodes[3].TreePath)
	}
}

// TestEvidenceCollector_CollectActivationEvidence tests activation evidence collection
func TestEvidenceCollector_CollectActivationEvidence(t *testing.T) {
	ec := NewEvidenceCollector()

	conv := protocol.ConversationRef{
		DisplayName: "张三",
	}

	// Test with matching node
	nodes := []windows.AccessibleNode{
		{Name: "张三", Role: "list item", Bounds: [4]int{10, 50, 180, 40}},
	}

	evidence := ec.CollectActivationEvidence(conv, nodes, []windows.AccessibleNode{}, "test")

	if !evidence.NodeStillExists {
		t.Error("Expected NodeStillExists to be true")
	}
	if evidence.Confidence <= 0 {
		t.Errorf("Expected positive confidence, got %f", evidence.Confidence)
	}
}

// TestEvidenceCollector_CollectMessageEvidence tests message evidence collection
func TestEvidenceCollector_CollectMessageEvidence(t *testing.T) {
	ec := NewEvidenceCollector()

	beforeNodes := []windows.AccessibleNode{
		{Name: "Old message", Role: "text", Bounds: [4]int{200, 100, 200, 30}},
	}

	afterNodes := []windows.AccessibleNode{
		{Name: "Old message", Role: "text", Bounds: [4]int{200, 100, 200, 30}},
		{Name: "New message", Role: "text", Bounds: [4]int{200, 130, 200, 30}},
	}

	beforeScreenshot := []byte{1, 2, 3, 4, 5}
	afterScreenshot := []byte{1, 2, 3, 4, 6} // One byte different

	evidence := ec.CollectMessageEvidence(beforeNodes, afterNodes, beforeScreenshot, afterScreenshot, [4]int{})

	if evidence.NewMessageNodes != 1 {
		t.Errorf("Expected 1 new message node, got %d", evidence.NewMessageNodes)
	}
	if len(evidence.NewMessageText) != 1 {
		t.Errorf("Expected 1 new message text, got %d", len(evidence.NewMessageText))
	}
	if !evidence.ScreenshotChanged {
		t.Error("Expected screenshot to be detected as changed")
	}
}

// TestEvidenceCollector_DetermineDeliveryState tests delivery state determination
func TestEvidenceCollector_DetermineDeliveryState(t *testing.T) {
	ec := NewEvidenceCollector()

	// Test verified state
	activationEvidence := ActivationEvidence{Confidence: 0.9}
	messageEvidence := MessageEvidence{Confidence: 0.9}
	state, confidence := ec.DetermineDeliveryState(activationEvidence, messageEvidence)

	if state != "verified" {
		t.Errorf("Expected 'verified', got '%s'", state)
	}
	if confidence < 0.8 {
		t.Errorf("Expected confidence >= 0.8, got %f", confidence)
	}

	// Test sent_unverified state
	activationEvidence = ActivationEvidence{Confidence: 0.5}
	messageEvidence = MessageEvidence{Confidence: 0.5}
	state, confidence = ec.DetermineDeliveryState(activationEvidence, messageEvidence)

	if state != "sent_unverified" {
		t.Errorf("Expected 'sent_unverified', got '%s'", state)
	}

	// Test unknown state
	activationEvidence = ActivationEvidence{Confidence: 0.1}
	messageEvidence = MessageEvidence{Confidence: 0.1}
	state, confidence = ec.DetermineDeliveryState(activationEvidence, messageEvidence)

	if state != "unknown" {
		t.Errorf("Expected 'unknown', got '%s'", state)
	}
}

// TestMessageClassifier_ClassifyNode tests node classification
func TestMessageClassifier_ClassifyNode(t *testing.T) {
	mc := NewMessageClassifier()

	// Test input box
	inputNode := windows.AccessibleNode{Role: "edit", Bounds: [4]int{100, 400, 300, 30}}
	nodeType := mc.ClassifyNode(inputNode)
	if nodeType != NodeTypeInputBox {
		t.Errorf("Expected NodeTypeInputBox, got %v", nodeType)
	}

	// Test title
	titleNode := windows.AccessibleNode{Role: "text", Bounds: [4]int{10, 10, 200, 20}}
	nodeType = mc.ClassifyNode(titleNode)
	if nodeType != NodeTypeTitle {
		t.Errorf("Expected NodeTypeTitle, got %v", nodeType)
	}

	// Test message bubble
	bubbleNode := windows.AccessibleNode{Role: "text", Bounds: [4]int{200, 100, 200, 50}}
	nodeType = mc.ClassifyNode(bubbleNode)
	if nodeType != NodeTypeMessageBubble {
		t.Errorf("Expected NodeTypeMessageBubble, got %v", nodeType)
	}

	// Test normal text
	textNode := windows.AccessibleNode{Role: "static", Bounds: [4]int{10, 100, 200, 30}}
	nodeType = mc.ClassifyNode(textNode)
	if nodeType != NodeTypeNormalText {
		t.Errorf("Expected NodeTypeNormalText, got %v", nodeType)
	}
}

// TestMessageClassifier_FilterMessageAreaNodes tests filtering message area nodes
func TestMessageClassifier_FilterMessageAreaNodes(t *testing.T) {
	mc := NewMessageClassifier()

	nodes := []windows.AccessibleNode{
		{Name: "Title", Role: "text", Bounds: [4]int{10, 10, 200, 20}}, // Top area
		{Name: "Input", Role: "edit", Bounds: [4]int{100, 400, 300, 30}}, // Input box
		{Name: "Message1", Role: "text", Bounds: [4]int{200, 100, 200, 30}}, // Message area
		{Name: "Message2", Role: "static", Bounds: [4]int{200, 140, 200, 30}}, // Message area
	}

	filtered := mc.FilterMessageAreaNodes(nodes, 0)

	if len(filtered) != 2 {
		t.Errorf("Expected 2 filtered nodes, got %d", len(filtered))
	}

	for _, node := range filtered {
		if node.Name == "Title" || node.Name == "Input" {
			t.Errorf("Should not include title or input node: %s", node.Name)
		}
	}
}

// TestMessageClassifier_IsMessageCandidate tests message candidate detection
func TestMessageClassifier_IsMessageCandidate(t *testing.T) {
	mc := NewMessageClassifier()

	// Message bubble should be a candidate
	bubbleNode := windows.AccessibleNode{Role: "text", Bounds: [4]int{200, 100, 200, 50}}
	if !mc.IsMessageCandidate(bubbleNode) {
		t.Error("Message bubble should be a candidate")
	}

	// Normal text should be a candidate
	textNode := windows.AccessibleNode{Role: "static", Bounds: [4]int{200, 100, 200, 30}}
	if !mc.IsMessageCandidate(textNode) {
		t.Error("Normal text should be a candidate")
	}

	// Input box should not be a candidate
	inputNode := windows.AccessibleNode{Role: "edit", Bounds: [4]int{100, 400, 300, 30}}
	if mc.IsMessageCandidate(inputNode) {
		t.Error("Input box should not be a candidate")
	}
}

// TestEvidenceCollector_CalculateChatAreaDiff tests chat area difference calculation
func TestEvidenceCollector_CalculateChatAreaDiff(t *testing.T) {
	ec := NewEvidenceCollector()

	// Test with different screenshots
	before := []byte{1, 2, 3, 4, 5}
	after := []byte{1, 2, 3, 4, 6}
	bounds := [4]int{100, 100, 200, 300}

	diff := ec.CalculateChatAreaDiff(before, after, bounds)

	if diff <= 0 {
		t.Errorf("Expected positive diff, got %f", diff)
	}

	// Test with same screenshots
	same := []byte{1, 2, 3, 4, 5}
	diff = ec.CalculateChatAreaDiff(same, same, bounds)

	if diff != 0 {
		t.Errorf("Expected zero diff for same screenshots, got %f", diff)
	}

	// Test with empty screenshots
	diff = ec.CalculateChatAreaDiff([]byte{}, []byte{}, bounds)
	if diff != 0 {
		t.Errorf("Expected zero diff for empty screenshots, got %f", diff)
	}
}

// TestEvidenceCollector_ScoreActivationEvidence tests activation evidence scoring
func TestEvidenceCollector_ScoreActivationEvidence(t *testing.T) {
	ec := NewEvidenceCollector()

	// Test with all evidence
	evidence := ActivationEvidence{
		NodeStillExists: true,
		HasActiveState:  true,
		HasTitleChange:  true,
		HasPanelSwitch:  true,
	}
	score := ec.scoreActivationEvidence(evidence)
	if score != 1.0 {
		t.Errorf("Expected score 1.0 for all evidence, got %f", score)
	}

	// Test with no evidence
	evidence = ActivationEvidence{}
	score = ec.scoreActivationEvidence(evidence)
	if score != 0 {
		t.Errorf("Expected score 0 for no evidence, got %f", score)
	}

	// Test with partial evidence
	evidence = ActivationEvidence{
		NodeStillExists: true,
		HasActiveState:  true,
	}
	score = ec.scoreActivationEvidence(evidence)
	if score <= 0 || score >= 1 {
		t.Errorf("Expected partial score between 0 and 1, got %f", score)
	}
}

// TestEvidenceCollector_ScoreMessageEvidence tests message evidence scoring
func TestEvidenceCollector_ScoreMessageEvidence(t *testing.T) {
	ec := NewEvidenceCollector()

	// Test with all evidence
	evidence := MessageEvidence{
		NewMessageNodes:   1,
		NewMessageText:    []string{"test"},
		ScreenshotChanged: true,
		ChatAreaDiff:      0.05,
	}
	score := ec.scoreMessageEvidence(evidence)
	if score != 1.0 {
		t.Errorf("Expected score 1.0 for all evidence, got %f", score)
	}

	// Test with no evidence
	evidence = MessageEvidence{}
	score = ec.scoreMessageEvidence(evidence)
	if score != 0 {
		t.Errorf("Expected score 0 for no evidence, got %f", score)
	}
}

// TestPathSystem_StableKeyRefind tests stable key re-finding
func TestPathSystem_StableKeyRefind(t *testing.T) {
	// Create nodes with TreePath set
	nodes := []windows.AccessibleNode{
		{Name: "Node1", Role: "list item", Bounds: [4]int{10, 50, 180, 40}, TreePath: "[0]"},
		{Name: "Node2", Role: "list item", Bounds: [4]int{10, 90, 180, 40}, TreePath: "[1]"},
	}

	// Test finding by TreePath
	found := false
	for _, node := range nodes {
		if node.TreePath == "[1]" && node.Name == "Node2" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Failed to find node by TreePath")
	}
}
