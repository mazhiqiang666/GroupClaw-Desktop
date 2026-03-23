package wechat

import (
	"testing"

	"github.com/yourorg/auto-customer-service/internal/agent/windows"
	"github.com/yourorg/auto-customer-service/pkg/protocol"
)

// ==================== High-Risk Regression Matrix Tests ====================
// These tests cover critical edge cases that could cause production issues

// TestRegression_SameNameContactMisSending tests the scenario where
// multiple contacts have the same name and message could be sent to wrong contact
func TestRegression_SameNameContactMisSending(t *testing.T) {
	pathSystem := NewPathSystem()
	rules := NewPositioningStrategyRules(pathSystem)

	// Two contacts with same name but different positions
	nodes := []windows.AccessibleNode{
		{
			Name:     "张三",
			Role:     "list item",
			Bounds:   [4]int{50, 100, 150, 40},
			TreePath: "[0]",
		},
		{
			Name:     "张三",
			Role:     "list item",
			Bounds:   [4]int{50, 200, 150, 40},
			TreePath: "[1]",
		},
	}

	// Test 1: First contact with tree path hint
	conv1 := protocol.ConversationRef{
		DisplayName:         "张三",
		ListNeighborhoodHint: []string{"[0]", "bounds:50_100_150_40"},
	}
	result1 := rules.FindNodeByStrategy(nodes, conv1)
	if result1.Node == nil || result1.Node.Name != "张三" {
		t.Errorf("Expected to find first '张三' by tree path, got %v", result1.Node)
	}
	if result1.Node.Bounds[1] != 100 {
		t.Errorf("Expected first contact at y=100, got y=%d", result1.Node.Bounds[1])
	}

	// Test 2: Second contact with different tree path hint
	conv2 := protocol.ConversationRef{
		DisplayName:         "张三",
		ListNeighborhoodHint: []string{"[1]", "bounds:50_200_150_40"},
	}
	result2 := rules.FindNodeByStrategy(nodes, conv2)
	if result2.Node == nil || result2.Node.Name != "张三" {
		t.Errorf("Expected to find second '张三' by tree path, got %v", result2.Node)
	}
	if result2.Node.Bounds[1] != 200 {
		t.Errorf("Expected second contact at y=200, got y=%d", result2.Node.Bounds[1])
	}

	// Test 3: Without hints, should find first match (potential mis-send risk)
	conv3 := protocol.ConversationRef{
		DisplayName: "张三",
	}
	result3 := rules.FindNodeByStrategy(nodes, conv3)
	if result3.Node == nil || result3.Node.Name != "张三" {
		t.Errorf("Expected to find '张三' by name match, got %v", result3.Node)
	}
	// This is the risk: without hints, we might pick the wrong contact
	if result3.Source != "name_match" {
		t.Errorf("Expected name_match source for ambiguous contact, got %s", result3.Source)
	}
}

// TestRegression_TitleSystemPromptMisIdentification tests the scenario where
// message classifier mis-identifies title/system prompt as normal text
func TestRegression_TitleSystemPromptMisIdentification(t *testing.T) {
	messageClassifier := NewMessageClassifier()

	tests := []struct {
		name         string
		nodeName     string
		nodeRole     string
		nodeBounds   [4]int
		expectedType NodeType
	}{
		{
			name:       "Title node",
			nodeName:   "张三",
			nodeRole:   "text",
			nodeBounds: [4]int{50, 10, 300, 25},
			expectedType: NodeTypeTitle,
		},
		{
			name:       "System prompt",
			nodeName:   "You are now connected",
			nodeRole:   "text",
			nodeBounds: [4]int{200, 50, 200, 20},
			expectedType: NodeTypeSystemPrompt,
		},
		{
			name:       "Normal message",
			nodeName:   "Hello world",
			nodeRole:   "text",
			nodeBounds: [4]int{200, 100, 200, 30},
			expectedType: NodeTypeMessageBubble,
		},
		{
			name:       "Input box",
			nodeName:   "",
			nodeRole:   "edit",
			nodeBounds: [4]int{200, 400, 300, 30},
			expectedType: NodeTypeInputBox,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := windows.AccessibleNode{
				Name:   tt.nodeName,
				Role:   tt.nodeRole,
				Bounds: tt.nodeBounds,
			}
			nodeType := messageClassifier.ClassifyNode(node)
			if nodeType != tt.expectedType {
				t.Errorf("Expected type %v, got %v", tt.expectedType, nodeType)
			}
		})
	}
}

// TestRegression_InputBoxMisIdentification tests the scenario where
// input box is mis-identified as normal text, causing send failures
func TestRegression_InputBoxMisIdentification(t *testing.T) {
	messageClassifier := NewMessageClassifier()

	tests := []struct {
		name         string
		node         windows.AccessibleNode
		shouldBeInput bool
	}{
		{
			name: "Clear input box",
			node: windows.AccessibleNode{
				Name:   "",
				Role:   "edit",
				Bounds: [4]int{200, 400, 300, 30},
			},
			shouldBeInput: true,
		},
		{
			name: "Input box with placeholder",
			node: windows.AccessibleNode{
				Name:   "Type a message...",
				Role:   "edit",
				Bounds: [4]int{200, 400, 300, 30},
			},
			shouldBeInput: true,
		},
		{
			name: "Text node with edit role (edge case)",
			node: windows.AccessibleNode{
				Name:   "Some text",
				Role:   "edit",
				Bounds: [4]int{200, 400, 300, 30},
			},
			shouldBeInput: true,
		},
		{
			name: "Normal text node (should not be input)",
			node: windows.AccessibleNode{
				Name:   "Message content",
				Role:   "text",
				Bounds: [4]int{200, 100, 200, 30},
			},
			shouldBeInput: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodeType := messageClassifier.ClassifyNode(tt.node)
			isInput := nodeType == NodeTypeInputBox
			if isInput != tt.shouldBeInput {
				t.Errorf("Expected isInput=%v, got isInput=%v (type=%v)", tt.shouldBeInput, isInput, nodeType)
			}
		})
	}
}

// TestRegression_PathChangeFallbackErrors tests the scenario where
// path changes cause fallback errors in positioning
func TestRegression_PathChangeFallbackErrors(t *testing.T) {
	pathSystem := NewPathSystem()
	rules := NewPositioningStrategyRules(pathSystem)

	// Node with tree path that will change
	nodes := []windows.AccessibleNode{
		{
			Name:     "张三",
			Role:     "list item",
			Bounds:   [4]int{50, 100, 150, 40},
			TreePath: "[0]",
		},
	}

	// Test 1: Original tree path works
	conv1 := protocol.ConversationRef{
		DisplayName:         "张三",
		ListNeighborhoodHint: []string{"[0]", "bounds:50_100_150_40"},
	}
	result1 := rules.FindNodeByStrategy(nodes, conv1)
	if result1.Node == nil {
		t.Error("Expected to find node with original tree path")
	}
	if result1.Source != "tree_path_name" {
		t.Errorf("Expected tree_path_name source, got %s", result1.Source)
	}

	// Test 2: Changed tree path should fallback to bounds match
	conv2 := protocol.ConversationRef{
		DisplayName:         "张三",
		ListNeighborhoodHint: []string{"[99]", "bounds:50_100_150_40"}, // Wrong tree path
	}
	result2 := rules.FindNodeByStrategy(nodes, conv2)
	if result2.Node == nil {
		t.Error("Expected to find node with fallback strategy")
	}
	if result2.Source != "bounds_match" {
		t.Errorf("Expected bounds_match fallback source, got %s", result2.Source)
	}

	// Test 3: Both tree path and bounds changed - should fallback to name match
	conv3 := protocol.ConversationRef{
		DisplayName:         "张三",
		ListNeighborhoodHint: []string{"[99]", "bounds:999_999_999_999"},
	}
	result3 := rules.FindNodeByStrategy(nodes, conv3)
	if result3.Node == nil {
		t.Error("Expected to find node with name match fallback")
	}
	if result3.Source != "name_match" {
		t.Errorf("Expected name_match fallback source, got %s", result3.Source)
	}
}

// TestRegression_BoundsDriftClickErrors tests the scenario where
// bounds drift causes click position errors
func TestRegression_BoundsDriftClickErrors(t *testing.T) {
	pathSystem := NewPathSystem()
	rules := NewPositioningStrategyRules(pathSystem)

	tests := []struct {
		name      string
		node      *windows.AccessibleNode
		expectOk  bool
		expectX   int
		expectY   int
		tolerance int // Allowable tolerance for drift
	}{
		{
			name: "Normal bounds",
			node: &windows.AccessibleNode{
				Name:   "Test",
				Bounds: [4]int{100, 200, 200, 100},
			},
			expectOk:  true,
			expectX:   200, // 100 + 200/2
			expectY:   250, // 200 + 100/2
			tolerance: 0,
		},
		{
			name: "Slight drift in X position",
			node: &windows.AccessibleNode{
				Name:   "Test",
				Bounds: [4]int{102, 200, 200, 100}, // +2 drift
			},
			expectOk:  true,
			expectX:   202, // 102 + 200/2
			expectY:   250,
			tolerance: 5,
		},
		{
			name: "Slight drift in Y position",
			node: &windows.AccessibleNode{
				Name:   "Test",
				Bounds: [4]int{100, 202, 200, 100}, // +2 drift
			},
			expectOk:  true,
			expectX:   200,
			expectY:   252, // 202 + 100/2
			tolerance: 5,
		},
		{
			name: "Width drift",
			node: &windows.AccessibleNode{
				Name:   "Test",
				Bounds: [4]int{100, 200, 202, 100}, // +2 width drift
			},
			expectOk:  true,
			expectX:   201, // 100 + 202/2
			expectY:   250,
			tolerance: 2,
		},
		{
			name: "Height drift",
			node: &windows.AccessibleNode{
				Name:   "Test",
				Bounds: [4]int{100, 200, 200, 102}, // +2 height drift
			},
			expectOk:  true,
			expectX:   200,
			expectY:   251, // 200 + 102/2
			tolerance: 2,
		},
		{
			name: "Zero width (invalid)",
			node: &windows.AccessibleNode{
				Name:   "Test",
				Bounds: [4]int{100, 200, 0, 100},
			},
			expectOk: false,
		},
		{
			name: "Zero height (invalid)",
			node: &windows.AccessibleNode{
				Name:   "Test",
				Bounds: [4]int{100, 200, 200, 0},
			},
			expectOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x, y, ok := rules.CalculateClickPosition(tt.node)

			if ok != tt.expectOk {
				t.Errorf("Expected ok=%v, got ok=%v", tt.expectOk, ok)
			}

			if ok && tt.expectOk {
				if tt.tolerance == 0 {
					if x != tt.expectX {
						t.Errorf("Expected x=%d, got x=%d", tt.expectX, x)
					}
					if y != tt.expectY {
						t.Errorf("Expected y=%d, got y=%d", tt.expectY, y)
					}
				} else {
					// Allow tolerance for drift scenarios
					if abs(x-tt.expectX) > tt.tolerance {
						t.Errorf("Expected x within %d of %d, got %d", tt.tolerance, tt.expectX, x)
					}
					if abs(y-tt.expectY) > tt.tolerance {
						t.Errorf("Expected y within %d of %d, got %d", tt.tolerance, tt.expectY, y)
					}
				}
			}
		})
	}
}

// TestRegression_DeliveryStateMisJudgment tests the scenario where
// delivery state is incorrectly determined
func TestRegression_DeliveryStateMisJudgment(t *testing.T) {
	rules := NewDeliveryAssessmentRules()

	tests := []struct {
		name             string
		focusEvidence    FocusVerificationEvidence
		messageEvidence  SendVerificationEvidence
		expectedState    string
		expectedMinConf  float64
		expectedMaxConf  float64
	}{
		{
			name: "High confidence verified",
			focusEvidence: FocusVerificationEvidence{
				NodeStillExists:    true,
				NodeHasActiveState: true,
				Confidence:         0.95,
				EvidenceCount:      3,
			},
			messageEvidence: SendVerificationEvidence{
				NewMessageNodes:   1,
				MessageNodeAdded:  true,
				MessageContentMatch: true,
				ScreenshotChanged: true,
				Confidence:        0.95,
			},
			expectedState:   "verified",
			expectedMinConf: 0.8,
			expectedMaxConf: 1.0,
		},
		{
			name: "Borderline verified (0.79)",
			focusEvidence: FocusVerificationEvidence{
				NodeStillExists:    true,
				NodeHasActiveState: true,
				Confidence:         0.79,
				EvidenceCount:      2,
			},
			messageEvidence: SendVerificationEvidence{
				NewMessageNodes:   1,
				MessageNodeAdded:  true,
				MessageContentMatch: true,
				ScreenshotChanged: true,
				Confidence:        0.79,
			},
			expectedState:   "sent_unverified",
			expectedMinConf: 0.5,
			expectedMaxConf: 0.8,
		},
		{
			name: "Exactly at verified threshold (0.8)",
			focusEvidence: FocusVerificationEvidence{
				NodeStillExists:    true,
				NodeHasActiveState: true,
				Confidence:         0.8,
				EvidenceCount:      2,
			},
			messageEvidence: SendVerificationEvidence{
				NewMessageNodes:   1,
				MessageNodeAdded:  true,
				MessageContentMatch: true,
				ScreenshotChanged: true,
				Confidence:        0.8,
			},
			expectedState:   "verified",
			expectedMinConf: 0.8,
			expectedMaxConf: 0.8,
		},
		{
			name: "Low confidence unknown",
			focusEvidence: FocusVerificationEvidence{
				NodeStillExists:    false,
				NodeHasActiveState: false,
				Confidence:         0.3,
				EvidenceCount:      1,
			},
			messageEvidence: SendVerificationEvidence{
				NewMessageNodes:   0,
				MessageNodeAdded:  false,
				MessageContentMatch: false,
				ScreenshotChanged: false,
				Confidence:        0.3,
			},
			expectedState:   "unknown",
			expectedMinConf: 0.0,
			expectedMaxConf: 0.5,
		},
		{
			name: "Focus only - verified",
			focusEvidence: FocusVerificationEvidence{
				NodeStillExists:    true,
				NodeHasActiveState: true,
				Confidence:         0.9,
				EvidenceCount:      2,
			},
			messageEvidence: SendVerificationEvidence{}, // Empty message evidence
			expectedState:   "verified",
			expectedMinConf: 0.8,
			expectedMaxConf: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assessment := rules.AssessDeliveryState(tt.focusEvidence, tt.messageEvidence)

			if assessment.State != tt.expectedState {
				t.Errorf("Expected state '%s', got '%s'", tt.expectedState, assessment.State)
			}

			if assessment.Confidence < tt.expectedMinConf || assessment.Confidence > tt.expectedMaxConf {
				t.Errorf("Confidence %f not in expected range [%f, %f]",
					assessment.Confidence, tt.expectedMinConf, tt.expectedMaxConf)
			}
		})
	}
}

// TestRegression_MultiContactSameNameWithHints tests the specific scenario
// where multiple contacts have same name but different hints should disambiguate
func TestRegression_MultiContactSameNameWithHints(t *testing.T) {
	pathSystem := NewPathSystem()
	rules := NewPositioningStrategyRules(pathSystem)

	// Three contacts all named "张三" at different positions
	nodes := []windows.AccessibleNode{
		{Name: "张三", Role: "list item", Bounds: [4]int{50, 100, 150, 40}, TreePath: "[0]"},
		{Name: "张三", Role: "list item", Bounds: [4]int{50, 150, 150, 40}, TreePath: "[1]"},
		{Name: "张三", Role: "list item", Bounds: [4]int{50, 200, 150, 40}, TreePath: "[2]"},
	}

	tests := []struct {
		name          string
		conv          protocol.ConversationRef
		expectedY     int
		expectedSrc   string
	}{
		{
			name: "First 张三 with tree path",
			conv: protocol.ConversationRef{
				DisplayName:         "张三",
				ListNeighborhoodHint: []string{"[0]", "bounds:50_100_150_40"},
			},
			expectedY:   100,
			expectedSrc: "tree_path_name",
		},
		{
			name: "Second 张三 with tree path",
			conv: protocol.ConversationRef{
				DisplayName:         "张三",
				ListNeighborhoodHint: []string{"[1]", "bounds:50_150_150_40"},
			},
			expectedY:   150,
			expectedSrc: "tree_path_name",
		},
		{
			name: "Third 张三 with tree path",
			conv: protocol.ConversationRef{
				DisplayName:         "张三",
				ListNeighborhoodHint: []string{"[2]", "bounds:50_200_150_40"},
			},
			expectedY:   200,
			expectedSrc: "tree_path_name",
		},
		{
			name: "First 张三 with bounds only",
			conv: protocol.ConversationRef{
				DisplayName:         "张三",
				ListNeighborhoodHint: []string{"[99]", "bounds:50_100_150_40"},
			},
			expectedY:   100,
			expectedSrc: "bounds_match",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rules.FindNodeByStrategy(nodes, tt.conv)

			if result.Node == nil {
				t.Error("Expected to find node")
				return
			}

			if result.Node.Bounds[1] != tt.expectedY {
				t.Errorf("Expected y=%d, got y=%d", tt.expectedY, result.Node.Bounds[1])
			}

			if result.Source != tt.expectedSrc {
				t.Errorf("Expected source '%s', got '%s'", tt.expectedSrc, result.Source)
			}
		})
	}
}

// Helper function for absolute value
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
