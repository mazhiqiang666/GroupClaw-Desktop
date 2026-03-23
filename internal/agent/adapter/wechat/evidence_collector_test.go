package wechat

import (
	"testing"

	"github.com/yourorg/auto-customer-service/internal/agent/windows"
	"github.com/yourorg/auto-customer-service/pkg/protocol"
)

// ==================== Evidence Collector Tests ====================

func TestEvidenceCollector_CollectActivationEvidence(t *testing.T) {
	ec := NewEvidenceCollector()

	conv := protocol.ConversationRef{
		DisplayName: "张三",
	}

	tests := []struct {
		name                 string
		nodes                []windows.AccessibleNode
		originalNodes        []windows.AccessibleNode
		locateSource         string
		expectExists         bool
		expectActive         bool
		expectTitle          bool
		expectPanel          bool
		expectMsgArea        bool
		expectZeroConfidence bool
	}{
		{
			name: "Full activation evidence",
			nodes: []windows.AccessibleNode{
				{Name: "张三", Role: "selected", Bounds: [4]int{50, 100, 150, 40}},
				{Name: "消息区域", Role: "text", Bounds: [4]int{300, 200, 200, 50}},
			},
			originalNodes: []windows.AccessibleNode{
				{Name: "李四", Role: "list item", Bounds: [4]int{50, 100, 150, 40}},
			},
			locateSource:  "tree_path_name",
			expectExists:  true,
			expectActive:  true,
			expectTitle:   true, // Bounds[0] = 50 < 200 triggers title change
			expectPanel:   true,
			expectMsgArea: true,
		},
		{
			name: "Node exists but not active",
			nodes: []windows.AccessibleNode{
				{Name: "张三", Role: "list item", Bounds: [4]int{300, 100, 150, 40}},
			},
			originalNodes: []windows.AccessibleNode{},
			locateSource:  "name_match",
			expectExists:  true,
			expectActive:  false,
			expectTitle:   false,
			expectPanel:   false,
			expectMsgArea: false,
		},
		{
			name: "Node not found",
			nodes: []windows.AccessibleNode{
				{Name: "李四", Role: "list item", Bounds: [4]int{50, 100, 150, 40}},
			},
			originalNodes: []windows.AccessibleNode{},
			locateSource:  "name_match",
			expectExists:  false,
			expectActive:  false,
			expectTitle:   false,
			expectPanel:   false,
			expectMsgArea: false,
			expectZeroConfidence: true, // No node found means zero confidence
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evidence := ec.CollectActivationEvidence(conv, tt.nodes, tt.originalNodes, tt.locateSource)

			if evidence.NodeStillExists != tt.expectExists {
				t.Errorf("NodeStillExists = %v, want %v", evidence.NodeStillExists, tt.expectExists)
			}
			if evidence.HasActiveState != tt.expectActive {
				t.Errorf("HasActiveState = %v, want %v", evidence.HasActiveState, tt.expectActive)
			}
			if evidence.HasTitleChange != tt.expectTitle {
				t.Errorf("HasTitleChange = %v, want %v", evidence.HasTitleChange, tt.expectTitle)
			}
			if evidence.HasPanelSwitch != tt.expectPanel {
				t.Errorf("HasPanelSwitch = %v, want %v", evidence.HasPanelSwitch, tt.expectPanel)
			}
			if evidence.LocateSource != tt.locateSource {
				t.Errorf("LocateSource = %s, want %s", evidence.LocateSource, tt.locateSource)
			}
			if tt.expectZeroConfidence {
				if evidence.Confidence != 0 {
					t.Errorf("Confidence should be 0, got %f", evidence.Confidence)
				}
			} else {
				if evidence.Confidence <= 0 {
					t.Errorf("Confidence should be positive, got %f", evidence.Confidence)
				}
			}
		})
	}
}

func TestEvidenceCollector_CollectMessageEvidence(t *testing.T) {
	ec := NewEvidenceCollector()

	tests := []struct {
		name                 string
		beforeNodes          []windows.AccessibleNode
		afterNodes           []windows.AccessibleNode
		beforeScreenshot     []byte
		afterScreenshot      []byte
		chatAreaBounds       [4]int
		expectNewNodes       int
		expectChanged        bool
		expectZeroConfidence bool
	}{
		{
			name: "New message detected",
			beforeNodes: []windows.AccessibleNode{
				{Name: "Old message", Role: "text", Bounds: [4]int{200, 100, 200, 30}},
			},
			afterNodes: []windows.AccessibleNode{
				{Name: "Old message", Role: "text", Bounds: [4]int{200, 100, 200, 30}},
				{Name: "New message", Role: "text", Bounds: [4]int{200, 130, 200, 30}},
			},
			beforeScreenshot: []byte{1, 2, 3, 4, 5},
			afterScreenshot:  []byte{1, 2, 3, 4, 6},
			chatAreaBounds:   [4]int{200, 100, 200, 30},
			expectNewNodes:   1,
			expectChanged:    true,
		},
		{
			name: "No new messages",
			beforeNodes: []windows.AccessibleNode{
				{Name: "Message", Role: "text", Bounds: [4]int{200, 100, 200, 30}},
			},
			afterNodes: []windows.AccessibleNode{
				{Name: "Message", Role: "text", Bounds: [4]int{200, 100, 200, 30}},
			},
			beforeScreenshot: []byte{1, 2, 3, 4, 5},
			afterScreenshot:  []byte{1, 2, 3, 4, 5},
			chatAreaBounds:   [4]int{200, 100, 200, 30},
			expectNewNodes:   0,
			expectChanged:    false,
			expectZeroConfidence: true, // No new messages means zero confidence
		},
		{
			name: "Multiple new messages",
			beforeNodes:      []windows.AccessibleNode{},
			afterNodes: []windows.AccessibleNode{
				{Name: "Msg1", Role: "text", Bounds: [4]int{200, 100, 200, 30}},
				{Name: "Msg2", Role: "text", Bounds: [4]int{200, 130, 200, 30}},
			},
			beforeScreenshot: []byte{1, 2, 3, 4, 5},
			afterScreenshot:  []byte{1, 2, 3, 4, 6},
			chatAreaBounds:   [4]int{200, 100, 200, 30},
			expectNewNodes:   2,
			expectChanged:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evidence := ec.CollectMessageEvidence(
				tt.beforeNodes, tt.afterNodes,
				tt.beforeScreenshot, tt.afterScreenshot,
				tt.chatAreaBounds,
			)

			if evidence.NewMessageNodes != tt.expectNewNodes {
				t.Errorf("NewMessageNodes = %d, want %d", evidence.NewMessageNodes, tt.expectNewNodes)
			}
			if evidence.ScreenshotChanged != tt.expectChanged {
				t.Errorf("ScreenshotChanged = %v, want %v", evidence.ScreenshotChanged, tt.expectChanged)
			}
			if tt.expectZeroConfidence {
				if evidence.Confidence != 0 {
					t.Errorf("Confidence should be 0, got %f", evidence.Confidence)
				}
			} else {
				if evidence.Confidence <= 0 {
					t.Errorf("Confidence should be positive, got %f", evidence.Confidence)
				}
			}
		})
	}
}

func TestEvidenceCollector_DetermineDeliveryState(t *testing.T) {
	ec := NewEvidenceCollector()

	tests := []struct {
		name                string
		activationEvidence  ActivationEvidence
		messageEvidence     MessageEvidence
		expectedState       string
		expectedMinConf     float64
	}{
		{
			name: "Verified state",
			activationEvidence: ActivationEvidence{
				HasActiveState:  true,
				NodeStillExists: true,
				Confidence:      0.9,
			},
			messageEvidence: MessageEvidence{
				NewMessageNodes: 1,
				Confidence:      0.9,
			},
			expectedState:   "verified",
			expectedMinConf: 0.8,
		},
		{
			name: "Sent unverified state",
			activationEvidence: ActivationEvidence{
				HasActiveState:  true,
				NodeStillExists: true,
				Confidence:      0.6,
			},
			messageEvidence: MessageEvidence{
				NewMessageNodes: 1,
				Confidence:      0.6,
			},
			expectedState:   "sent_unverified",
			expectedMinConf: 0.5,
		},
		{
			name: "Unknown state",
			activationEvidence: ActivationEvidence{
				HasActiveState:  false,
				NodeStillExists: false,
				Confidence:      0.2,
			},
			messageEvidence: MessageEvidence{
				NewMessageNodes: 0,
				Confidence:      0.2,
			},
			expectedState:   "unknown",
			expectedMinConf: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state, confidence := ec.DetermineDeliveryState(tt.activationEvidence, tt.messageEvidence)

			if state != tt.expectedState {
				t.Errorf("State = %s, want %s", state, tt.expectedState)
			}
			if confidence < tt.expectedMinConf {
				t.Errorf("Confidence = %f, want >= %f", confidence, tt.expectedMinConf)
			}
		})
	}
}

// ==================== Evidence Collector Dirty Data Tests ====================

func TestEvidenceCollector_CollectActivationEvidence_DirtyData(t *testing.T) {
	ec := NewEvidenceCollector()

	conv := protocol.ConversationRef{
		DisplayName: "张三",
	}

	tests := []struct {
		name          string
		nodes         []windows.AccessibleNode
		originalNodes []windows.AccessibleNode
	}{
		{
			name: "Empty nodes",
			nodes:         []windows.AccessibleNode{},
			originalNodes: []windows.AccessibleNode{},
		},
		{
			name: "Node with empty name",
			nodes: []windows.AccessibleNode{
				{Name: "", Role: "list item", Bounds: [4]int{50, 100, 150, 40}},
			},
			originalNodes: []windows.AccessibleNode{},
		},
		{
			name: "Node with invalid bounds",
			nodes: []windows.AccessibleNode{
				{Name: "张三", Role: "list item", Bounds: [4]int{50, 100, 0, 40}},
			},
			originalNodes: []windows.AccessibleNode{},
		},
		{
			name: "Unicode name",
			nodes: []windows.AccessibleNode{
				{Name: "测试用户", Role: "list item", Bounds: [4]int{50, 100, 150, 40}},
			},
			originalNodes: []windows.AccessibleNode{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			evidence := ec.CollectActivationEvidence(conv, tt.nodes, tt.originalNodes, "test")

			// Confidence should always be calculated
			if evidence.Confidence < 0 || evidence.Confidence > 1 {
				t.Errorf("Confidence out of range: %f", evidence.Confidence)
			}
		})
	}
}

func TestEvidenceCollector_CollectMessageEvidence_DirtyData(t *testing.T) {
	ec := NewEvidenceCollector()

	tests := []struct {
		name             string
		beforeNodes      []windows.AccessibleNode
		afterNodes       []windows.AccessibleNode
		beforeScreenshot []byte
		afterScreenshot  []byte
		chatAreaBounds   [4]int
	}{
		{
			name:             "Empty screenshots",
			beforeNodes:      []windows.AccessibleNode{},
			afterNodes:       []windows.AccessibleNode{},
			beforeScreenshot: []byte{},
			afterScreenshot:  []byte{},
			chatAreaBounds:   [4]int{},
		},
		{
			name: "Node with empty name",
			beforeNodes: []windows.AccessibleNode{},
			afterNodes: []windows.AccessibleNode{
				{Name: "", Role: "text", Bounds: [4]int{200, 100, 200, 30}},
			},
			beforeScreenshot: []byte{1, 2, 3},
			afterScreenshot:  []byte{1, 2, 4},
			chatAreaBounds:   [4]int{200, 100, 200, 30},
		},
		{
			name: "Invalid bounds",
			beforeNodes: []windows.AccessibleNode{},
			afterNodes: []windows.AccessibleNode{
				{Name: "Test", Role: "text", Bounds: [4]int{200, 100, 0, 30}},
			},
			beforeScreenshot: []byte{1, 2, 3},
			afterScreenshot:  []byte{1, 2, 4},
			chatAreaBounds:   [4]int{200, 100, 200, 30},
		},
		{
			name: "Different screenshot lengths",
			beforeNodes: []windows.AccessibleNode{},
			afterNodes: []windows.AccessibleNode{
				{Name: "Test", Role: "text", Bounds: [4]int{200, 100, 200, 30}},
			},
			beforeScreenshot: []byte{1, 2, 3},
			afterScreenshot:  []byte{1, 2, 3, 4, 5},
			chatAreaBounds:   [4]int{200, 100, 200, 30},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			evidence := ec.CollectMessageEvidence(
				tt.beforeNodes, tt.afterNodes,
				tt.beforeScreenshot, tt.afterScreenshot,
				tt.chatAreaBounds,
			)

			// Confidence should always be calculated
			if evidence.Confidence < 0 || evidence.Confidence > 1 {
				t.Errorf("Confidence out of range: %f", evidence.Confidence)
			}
		})
	}
}

func TestEvidenceCollector_ScoreActivationEvidence(t *testing.T) {
	ec := NewEvidenceCollector()

	tests := []struct {
		name       string
		evidence   ActivationEvidence
		minScore   float64
		maxScore   float64
	}{
		{
			name: "All evidence present",
			evidence: ActivationEvidence{
				HasActiveState:  true,
				NodeStillExists: true,
				HasTitleChange:  true,
				HasPanelSwitch:  true,
			},
			minScore: 0.9,
			maxScore: 1.0,
		},
		{
			name: "No evidence",
			evidence: ActivationEvidence{
				HasActiveState:  false,
				NodeStillExists: false,
				HasTitleChange:  false,
				HasPanelSwitch:  false,
			},
			minScore: 0.0,
			maxScore: 0.0,
		},
		{
			name: "Partial evidence",
			evidence: ActivationEvidence{
				HasActiveState:  true,
				NodeStillExists: true,
				HasTitleChange:  false,
				HasPanelSwitch:  false,
			},
			minScore: 0.4,
			maxScore: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := ec.scoreActivationEvidence(tt.evidence)
			// Use small epsilon for floating point comparison
			epsilon := 0.0001
			if score < tt.minScore-epsilon || score > tt.maxScore+epsilon {
				t.Errorf("Score = %f, want in range [%f, %f]", score, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestEvidenceCollector_ScoreMessageEvidence(t *testing.T) {
	ec := NewEvidenceCollector()

	tests := []struct {
		name     string
		evidence MessageEvidence
		minScore float64
		maxScore float64
	}{
		{
			name: "All evidence present",
			evidence: MessageEvidence{
				NewMessageNodes:   1,
				NewMessageText:    []string{"Hello"},
				ScreenshotChanged: true,
				ChatAreaDiff:      0.05,
			},
			minScore: 0.9,
			maxScore: 1.0,
		},
		{
			name: "No evidence",
			evidence: MessageEvidence{
				NewMessageNodes:   0,
				NewMessageText:    []string{},
				ScreenshotChanged: false,
				ChatAreaDiff:      0.0,
			},
			minScore: 0.0,
			maxScore: 0.0,
		},
		{
			name: "Partial evidence",
			evidence: MessageEvidence{
				NewMessageNodes:   1,
				NewMessageText:    []string{},
				ScreenshotChanged: false,
				ChatAreaDiff:      0.0,
			},
			minScore: 0.4,
			maxScore: 0.4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := ec.scoreMessageEvidence(tt.evidence)
			// Use small epsilon for floating point comparison
			epsilon := 0.0001
			if score < tt.minScore-epsilon || score > tt.maxScore+epsilon {
				t.Errorf("Score = %f, want in range [%f, %f]", score, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestEvidenceCollector_CalculateChatAreaDiff(t *testing.T) {
	ec := NewEvidenceCollector()

	tests := []struct {
		name           string
		before         []byte
		after          []byte
		chatAreaBounds [4]int
		expectDiff     float64
	}{
		{
			name:           "Identical screenshots",
			before:         []byte{1, 2, 3, 4, 5},
			after:          []byte{1, 2, 3, 4, 5},
			chatAreaBounds: [4]int{200, 100, 200, 30},
			expectDiff:     0.0,
		},
		{
			name:           "Completely different",
			before:         []byte{1, 1, 1, 1, 1},
			after:          []byte{2, 2, 2, 2, 2},
			chatAreaBounds: [4]int{200, 100, 200, 30},
			expectDiff:     1.0,
		},
		{
			name:           "Empty screenshots",
			before:         []byte{},
			after:          []byte{},
			chatAreaBounds: [4]int{200, 100, 200, 30},
			expectDiff:     0.0,
		},
		{
			name:           "Invalid bounds",
			before:         []byte{1, 2, 3},
			after:          []byte{1, 2, 4},
			chatAreaBounds: [4]int{200, 100, 0, 30},
			expectDiff:     0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := ec.CalculateChatAreaDiff(tt.before, tt.after, tt.chatAreaBounds)
			if diff != tt.expectDiff {
				t.Errorf("CalculateChatAreaDiff() = %f, want %f", diff, tt.expectDiff)
			}
		})
	}
}
