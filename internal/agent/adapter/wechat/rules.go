package wechat

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mazhiqiang666/GroupClaw-Desktop/internal/agent/windows"
	"github.com/mazhiqiang666/GroupClaw-Desktop/pkg/protocol"
)

// ==================== Structured Verification Results ====================

// FocusVerificationEvidence represents evidence for focus operation verification
type FocusVerificationEvidence struct {
	// Node-level evidence
	NodeStillExists   bool
	NodeHasActiveState bool
	NodeBoundsMatch   bool

	// Title-level evidence
	TitleContainsTarget bool
	TitleChanged        bool

	// Panel-level evidence
	PanelSwitchDetected bool
	MessageAreaVisible  bool

	// Confidence and source
	Confidence    float64
	LocateSource  string
	EvidenceCount int
}

// SendVerificationEvidence represents evidence for send operation verification
type SendVerificationEvidence struct {
	// Node-level evidence
	NewMessageNodes   int
	MessageNodeAdded  bool
	MessageContentMatch bool

	// Screenshot-level evidence
	ScreenshotChanged bool
	ChatAreaDiff      float64

	// Input box verification
	InputBoxClicked           bool
	InputBoxDiffAfterClick    float64
	InputBoxDiffAfterPaste    float64
	SuspectedInputNotFocused  bool
	SuspectedPasteFailed      bool

	// Message area verification (multiple time points)
	ChatAreaDiff300ms         float64
	ChatAreaDiff800ms         float64
	ChatAreaDiff1500ms        float64

	// Failure indicator detection
	FailureIndicatorFound    bool
	FailureIndicatorBbox     [4]int

	// Send failure detection
	SuspectedSendFailed bool
	SendFailureReason   string
	FailureSignalSource string

	// Confidence
	Confidence float64
}

// DeliveryAssessment represents the final delivery assessment
type DeliveryAssessment struct {
	State      string
	Confidence float64
	Evidence   FocusVerificationEvidence
	Messages   []string
}

// ==================== Session Candidate Rules ====================

// SessionCandidateRules handles candidate session identification
type SessionCandidateRules struct{}

// NewSessionCandidateRules creates a new session candidate rules instance
func NewSessionCandidateRules() *SessionCandidateRules {
	return &SessionCandidateRules{}
}

// IsCandidateConversation determines if a node represents a candidate conversation
func (r *SessionCandidateRules) IsCandidateConversation(node windows.AccessibleNode, windowWidth int) bool {
	// Rule 1: Check name is non-empty (most important)
	if node.Name == "" {
		return false
	}

	// Rule 2: Check bounds are valid (but allow some flexibility)
	if len(node.Bounds) != 4 {
		return false
	}
	bounds := node.Bounds
	// Check for negative or zero dimensions (but allow slightly negative positions)
	if bounds[0] < -50 || bounds[1] < -50 || bounds[2] <= 0 || bounds[3] <= 0 {
		return false
	}

	// Rule 3: Accept a wider range of roles for real WeChat
	acceptableRoles := map[string]bool{
		"list item": true,
		"ListItem": true,
		"text": true,
		"static": true,
		"client": true,
		"pane": true,
		"window": true,
		"": true, // Allow empty role for debugging
	}
	if !acceptableRoles[node.Role] {
		// Still allow if name looks like a contact name (contains non-ASCII or spaces)
		hasNonASCII := false
		for _, r := range node.Name {
			if r > 127 {
				hasNonASCII = true
				break
			}
		}
		if !hasNonASCII && len(node.Name) < 2 {
			return false
		}
	}

	return true
}

// FilterCandidateConversations filters nodes to only include candidate conversations
func (r *SessionCandidateRules) FilterCandidateConversations(
	nodes []windows.AccessibleNode,
	windowWidth int,
) []windows.AccessibleNode {
	var candidates []windows.AccessibleNode
	for _, node := range nodes {
		if r.IsCandidateConversation(node, windowWidth) {
			candidates = append(candidates, node)
		}
	}
	return candidates
}

// ==================== Positioning Strategy Rules ====================

// PositioningStrategyRules handles multi-strategy node positioning
type PositioningStrategyRules struct {
	pathSystem *PathSystem
}

// NewPositioningStrategyRules creates a new positioning strategy rules instance
func NewPositioningStrategyRules(pathSystem *PathSystem) *PositioningStrategyRules {
	return &PositioningStrategyRules{
		pathSystem: pathSystem,
	}
}

// PositioningResult represents the result of a positioning attempt
type PositioningResult struct {
	Node         *windows.AccessibleNode
	Source       string
	Confidence   float64
	ClickX       int
	ClickY       int
}

// FindNodeByStrategy attempts to find a node using multiple strategies
func (r *PositioningStrategyRules) FindNodeByStrategy(
	flatNodes []windows.AccessibleNode,
	conv protocol.ConversationRef,
) PositioningResult {
	// Strategy 1: Tree path + name match
	if len(conv.ListNeighborhoodHint) > 0 {
		treePath := conv.ListNeighborhoodHint[0]
		for i := range flatNodes {
			node := &flatNodes[i]
			if node.TreePath == treePath && node.Name == conv.DisplayName {
				return PositioningResult{
					Node:       node,
					Source:     "tree_path_name",
					Confidence: 1.0,
				}
			}
		}
	}

	// Strategy 2: Bounds match (from ListNeighborhoodHint)
	if len(conv.ListNeighborhoodHint) > 1 {
		boundsStr := conv.ListNeighborhoodHint[1]
		if strings.HasPrefix(boundsStr, "bounds:") {
			boundsStr = strings.TrimPrefix(boundsStr, "bounds:")
			boundsParts := strings.Split(boundsStr, "_")
			if len(boundsParts) == 4 {
				expectedBounds := [4]int{}
				for j := 0; j < 4; j++ {
					val, _ := strconv.Atoi(boundsParts[j])
					expectedBounds[j] = val
				}

				for i := range flatNodes {
					node := &flatNodes[i]
					if node.Name == conv.DisplayName &&
						len(node.Bounds) == 4 &&
						node.Bounds[0] == expectedBounds[0] &&
						node.Bounds[1] == expectedBounds[1] &&
						node.Bounds[2] == expectedBounds[2] &&
						node.Bounds[3] == expectedBounds[3] {
						return PositioningResult{
							Node:       node,
							Source:     "bounds_match",
							Confidence: 1.0,
						}
					}
				}
			}
		}
	}

	// Strategy 3: Stable key match (PreviewText)
	if conv.PreviewText != "" {
		for i := range flatNodes {
			node := &flatNodes[i]
			// Reconstruct stable key and compare
			key := r.generateStableKey(*node, "", node.TreePath)
			if key == conv.PreviewText {
				return PositioningResult{
					Node:       node,
					Source:     "stable_key",
					Confidence: 1.0,
				}
			}
		}
	}

	// Strategy 4: Name match only
	for i := range flatNodes {
		node := &flatNodes[i]
		if node.Name == conv.DisplayName &&
			(node.Role == "list item" || node.Role == "ListItem") {
			return PositioningResult{
				Node:       node,
				Source:     "name_match",
				Confidence: 0.8,
			}
		}
	}

	// No match found
	return PositioningResult{
		Node:       nil,
		Source:     "not_found",
		Confidence: 0.0,
	}
}

// CalculateClickPosition calculates the click position based on node bounds
func (r *PositioningStrategyRules) CalculateClickPosition(node *windows.AccessibleNode) (int, int, bool) {
	if node == nil || len(node.Bounds) != 4 {
		return 0, 0, false
	}

	bounds := node.Bounds
	// Validate bounds: width and height must be positive
	if bounds[2] <= 0 || bounds[3] <= 0 {
		return 0, 0, false
	}

	clickX := bounds[0] + bounds[2]/2 // center x
	clickY := bounds[1] + bounds[3]/2 // center y
	return clickX, clickY, true
}

// generateStableKey generates a stable定位 key for a node
func (r *PositioningStrategyRules) generateStableKey(node windows.AccessibleNode, parentContext string, treePath string) string {
	if len(node.Bounds) != 4 {
		return ""
	}
	// Format: tree_path|parent|role|name|x_y_w_h
	key := fmt.Sprintf("%s|%s|%s|%s|%d_%d_%d_%d",
		treePath, parentContext, node.Role, node.Name,
		node.Bounds[0], node.Bounds[1], node.Bounds[2], node.Bounds[3])
	return key
}

// ==================== Activation Verification Rules ====================

// ActivationVerificationRules handles session activation verification
type ActivationVerificationRules struct {
	pathSystem        *PathSystem
	evidenceCollector *EvidenceCollector
}

// NewActivationVerificationRules creates a new activation verification rules instance
func NewActivationVerificationRules(
	pathSystem *PathSystem,
	evidenceCollector *EvidenceCollector,
) *ActivationVerificationRules {
	return &ActivationVerificationRules{
		pathSystem:        pathSystem,
		evidenceCollector: evidenceCollector,
	}
}

// VerifySessionActivation verifies if a session is activated using multiple evidence sources
func (r *ActivationVerificationRules) VerifySessionActivation(
	conv protocol.ConversationRef,
	currentNodes []windows.AccessibleNode,
	originalNodes []windows.AccessibleNode,
	locateSource string,
) FocusVerificationEvidence {
	// Flatten current nodes
	flatNodes := r.pathSystem.FlattenNodesWithPath(currentNodes, "", 0, 10)

	evidence := FocusVerificationEvidence{
		LocateSource: locateSource,
	}

	// Evidence 1: Check if target node exists and has active state
	for i := range flatNodes {
		node := &flatNodes[i]
		if node.Name == conv.DisplayName &&
			(node.Role == "list item" || node.Role == "ListItem") {
			evidence.NodeStillExists = true

			// Check if in left list area (potential active state indicator)
			if len(node.Bounds) == 4 && node.Bounds[0] < 200 {
				evidence.NodeHasActiveState = true
			}
			break
		}
	}

	// Evidence 2: Check title contains target name
	for i := range flatNodes {
		node := &flatNodes[i]
		if node.Name != "" && strings.Contains(node.Name, conv.DisplayName) {
			// Check if it's in title area (top of window)
			if len(node.Bounds) == 4 && node.Bounds[1] < 50 {
				evidence.TitleContainsTarget = true
				break
			}
		}
	}

	// Evidence 3: Check panel switch (node count change)
	if len(originalNodes) > 0 && len(currentNodes) > 0 {
		ratio := float64(len(currentNodes)) / float64(len(originalNodes))
		if ratio < 0.8 || ratio > 1.2 {
			evidence.PanelSwitchDetected = true
		}
	}

	// Evidence 4: Check message area visibility
	for i := range flatNodes {
		node := &flatNodes[i]
		if len(node.Bounds) == 4 {
			windowWidth := node.Bounds[0] + node.Bounds[2]
			// Message area is typically on the right side
			if node.Bounds[0] > windowWidth/3 {
				if node.Role == "text" || node.Role == "static" {
					evidence.MessageAreaVisible = true
					break
				}
			}
		}
	}

	// Calculate confidence based on evidence
	evidence.EvidenceCount = 0
	if evidence.NodeStillExists {
		evidence.EvidenceCount++
	}
	if evidence.NodeHasActiveState {
		evidence.EvidenceCount++
	}
	if evidence.TitleContainsTarget {
		evidence.EvidenceCount++
	}
	if evidence.PanelSwitchDetected {
		evidence.EvidenceCount++
	}
	if evidence.MessageAreaVisible {
		evidence.EvidenceCount++
	}

	evidence.Confidence = r.calculateConfidence(evidence)

	return evidence
}

// calculateConfidence calculates confidence score based on evidence
func (r *ActivationVerificationRules) calculateConfidence(evidence FocusVerificationEvidence) float64 {
	score := 0.0
	totalWeight := 0.0

	// Node existence: 20%
	if evidence.NodeStillExists {
		score += 0.2
	}
	totalWeight += 0.2

	// Active state: 30%
	if evidence.NodeHasActiveState {
		score += 0.3
	}
	totalWeight += 0.3

	// Title change: 20%
	if evidence.TitleContainsTarget {
		score += 0.2
	}
	totalWeight += 0.2

	// Panel switch: 15%
	if evidence.PanelSwitchDetected {
		score += 0.15
	}
	totalWeight += 0.15

	// Message area visible: 15%
	if evidence.MessageAreaVisible {
		score += 0.15
	}
	totalWeight += 0.15

	if totalWeight == 0 {
		return 0
	}
	return score / totalWeight
}

// ==================== Message Verification Rules ====================

// MessageVerificationRules handles message send verification
type MessageVerificationRules struct {
	pathSystem        *PathSystem
	messageClassifier *MessageClassifier
	evidenceCollector *EvidenceCollector
}

// NewMessageVerificationRules creates a new message verification rules instance
func NewMessageVerificationRules(
	pathSystem *PathSystem,
	messageClassifier *MessageClassifier,
	evidenceCollector *EvidenceCollector,
) *MessageVerificationRules {
	return &MessageVerificationRules{
		pathSystem:        pathSystem,
		messageClassifier: messageClassifier,
		evidenceCollector: evidenceCollector,
	}
}

// VerifyMessageSend verifies if a message was sent successfully
func (r *MessageVerificationRules) VerifyMessageSend(
	beforeNodes []windows.AccessibleNode,
	afterNodes []windows.AccessibleNode,
	beforeScreenshot []byte,
	afterScreenshot []byte,
	chatAreaBounds [4]int,
	content string,
) SendVerificationEvidence {
	evidence := SendVerificationEvidence{}

	// Flatten nodes
	beforeFlat := r.pathSystem.FlattenNodesWithPath(beforeNodes, "", 0, 10)
	afterFlat := r.pathSystem.FlattenNodesWithPath(afterNodes, "", 0, 10)

	// Filter message area nodes
	beforeMessageNodes := r.messageClassifier.FilterMessageAreaNodes(beforeFlat, 0)
	afterMessageNodes := r.messageClassifier.FilterMessageAreaNodes(afterFlat, 0)

	// Evidence 1: New message nodes detected
	evidence.NewMessageNodes = len(afterMessageNodes) - len(beforeMessageNodes)
	if evidence.NewMessageNodes > 0 {
		evidence.MessageNodeAdded = true
	}

	// Evidence 2: Check if new node contains sent content
	for _, node := range afterMessageNodes {
		if node.Name != "" && content != "" && strings.Contains(node.Name, content) {
			evidence.MessageContentMatch = true
			break
		}
	}

	// Evidence 3: Screenshot change detection
	if len(beforeScreenshot) > 0 && len(afterScreenshot) > 0 {
		evidence.ScreenshotChanged = r.evidenceCollector.checkScreenshotChange(
			beforeScreenshot, afterScreenshot, chatAreaBounds)
		evidence.ChatAreaDiff = r.evidenceCollector.CalculateChatAreaDiff(
			beforeScreenshot, afterScreenshot, chatAreaBounds)
	}

	// Evidence 4: Send failure detection (basic heuristic)
	// If no new message nodes detected and low screenshot change, suspect send failure
	if evidence.NewMessageNodes <= 0 && evidence.ChatAreaDiff < 0.05 {
		evidence.SuspectedSendFailed = true
		evidence.SendFailureReason = "no_new_message_nodes_and_low_screenshot_change"
		evidence.FailureSignalSource = "basic_heuristic"
	} else {
		evidence.SuspectedSendFailed = false
		evidence.SendFailureReason = ""
		evidence.FailureSignalSource = ""
	}

	// Calculate confidence
	evidence.Confidence = r.calculateConfidence(evidence)

	return evidence
}

// VerifyMessageSendEnhanced verifies message send with enhanced differential analysis
func (r *MessageVerificationRules) VerifyMessageSendEnhanced(
	beforeNodes []windows.AccessibleNode,
	afterNodes []windows.AccessibleNode,
	beforeScreenshot []byte,
	afterScreenshot []byte,
	chatAreaBounds [4]int,
	content string,
	// Enhanced parameters
	inputBoxClicked bool,
	inputBoxRect windows.InputBoxRect,
	windowWidth int,
	windowHeight int,
	// Additional screenshots for multi-timepoint analysis
	beforeClickScreenshot []byte,   // 输入框点击前截图
	afterClickScreenshot []byte,    // 输入框点击后截图
	afterPasteScreenshot []byte,    // 粘贴后截图
	afterEnter300msScreenshot []byte, // Enter后300ms截图
	afterEnter800msScreenshot []byte, // Enter后800ms截图
	afterEnter1500msScreenshot []byte, // Enter后1500ms截图
) SendVerificationEvidence {
	evidence := SendVerificationEvidence{}
	evidence.InputBoxClicked = inputBoxClicked

	// Flatten nodes
	beforeFlat := r.pathSystem.FlattenNodesWithPath(beforeNodes, "", 0, 10)
	afterFlat := r.pathSystem.FlattenNodesWithPath(afterNodes, "", 0, 10)

	// Filter message area nodes
	beforeMessageNodes := r.messageClassifier.FilterMessageAreaNodes(beforeFlat, 0)
	afterMessageNodes := r.messageClassifier.FilterMessageAreaNodes(afterFlat, 0)

	// Evidence 1: New message nodes detected
	evidence.NewMessageNodes = len(afterMessageNodes) - len(beforeMessageNodes)
	if evidence.NewMessageNodes > 0 {
		evidence.MessageNodeAdded = true
	}

	// Evidence 2: Check if new node contains sent content
	for _, node := range afterMessageNodes {
		if node.Name != "" && content != "" && strings.Contains(node.Name, content) {
			evidence.MessageContentMatch = true
			break
		}
	}

	// Evidence 3: Input box verification (click and paste effects)
	if inputBoxRect.Width > 0 && inputBoxRect.Height > 0 {
		// Calculate input box diff after click
		if len(beforeClickScreenshot) > 0 && len(afterClickScreenshot) > 0 {
			evidence.InputBoxDiffAfterClick = windows.CalculateRectDiffPercent(
				beforeClickScreenshot, afterClickScreenshot,
				windowWidth, windowHeight, inputBoxRect,
			)
			// If click didn't cause change, suspect input not focused
			if evidence.InputBoxDiffAfterClick < 0.01 {
				evidence.SuspectedInputNotFocused = true
			}
		}
		// Calculate input box diff after paste
		if len(afterClickScreenshot) > 0 && len(afterPasteScreenshot) > 0 {
			evidence.InputBoxDiffAfterPaste = windows.CalculateRectDiffPercent(
				afterClickScreenshot, afterPasteScreenshot,
				windowWidth, windowHeight, inputBoxRect,
			)
			// If paste didn't cause change, suspect paste failed
			if evidence.InputBoxDiffAfterPaste < 0.01 {
				evidence.SuspectedPasteFailed = true
			}
		}
	}

	// Evidence 4: Multi-timepoint chat area diff analysis
	if len(beforeScreenshot) > 0 && len(afterEnter300msScreenshot) > 0 {
		evidence.ChatAreaDiff300ms = r.evidenceCollector.CalculateChatAreaDiff(
			beforeScreenshot, afterEnter300msScreenshot, chatAreaBounds)
	}
	if len(beforeScreenshot) > 0 && len(afterEnter800msScreenshot) > 0 {
		evidence.ChatAreaDiff800ms = r.evidenceCollector.CalculateChatAreaDiff(
			beforeScreenshot, afterEnter800msScreenshot, chatAreaBounds)
	}
	if len(beforeScreenshot) > 0 && len(afterEnter1500msScreenshot) > 0 {
		evidence.ChatAreaDiff1500ms = r.evidenceCollector.CalculateChatAreaDiff(
			beforeScreenshot, afterEnter1500msScreenshot, chatAreaBounds)
	}

	// Evidence 5: Screenshot change detection (traditional)
	if len(beforeScreenshot) > 0 && len(afterScreenshot) > 0 {
		evidence.ScreenshotChanged = r.evidenceCollector.checkScreenshotChange(
			beforeScreenshot, afterScreenshot, chatAreaBounds)
		evidence.ChatAreaDiff = r.evidenceCollector.CalculateChatAreaDiff(
			beforeScreenshot, afterScreenshot, chatAreaBounds)
	}

	// Evidence 6: Failure indicator detection
	if len(afterScreenshot) > 0 {
		failureFound, failureBbox := windows.DetectFailureIndicator(
			afterScreenshot, windowWidth, windowHeight, inputBoxRect, chatAreaBounds)
		evidence.FailureIndicatorFound = failureFound
		evidence.FailureIndicatorBbox = failureBbox
	}

	// Evidence 7: Send failure detection (enhanced heuristic)
	sendFailed := false
	failureReason := ""
	failureSource := ""

	// Rule 1: Input box not focused
	if evidence.SuspectedInputNotFocused {
		sendFailed = true
		failureReason = "input_box_not_activated"
		failureSource = "input_box_diff_low"
	}
	// Rule 2: Paste failed
	if evidence.SuspectedPasteFailed {
		sendFailed = true
		failureReason = "paste_failed"
		failureSource = "input_box_diff_low_after_paste"
	}
	// Rule 3: No message area change at any timepoint
	if evidence.ChatAreaDiff300ms < 0.01 && evidence.ChatAreaDiff800ms < 0.01 && evidence.ChatAreaDiff1500ms < 0.01 {
		sendFailed = true
		failureReason = "no_chat_area_change"
		failureSource = "multi_timepoint_analysis"
	}
	// Rule 4: Failure indicator found
	if evidence.FailureIndicatorFound {
		sendFailed = true
		failureReason = "failure_indicator_detected"
		failureSource = "visual_failure_detection"
	}
	// Rule 5: Original heuristic (fallback)
	if evidence.NewMessageNodes <= 0 && evidence.ChatAreaDiff < 0.05 {
		sendFailed = true
		failureReason = "no_new_message_nodes_and_low_screenshot_change"
		failureSource = "basic_heuristic"
	}

	evidence.SuspectedSendFailed = sendFailed
	evidence.SendFailureReason = failureReason
	evidence.FailureSignalSource = failureSource

	// Calculate confidence
	evidence.Confidence = r.calculateConfidenceEnhanced(evidence)

	return evidence
}

// calculateConfidenceEnhanced calculates confidence score with enhanced evidence
func (r *MessageVerificationRules) calculateConfidenceEnhanced(evidence SendVerificationEvidence) float64 {
	score := 0.0
	totalWeight := 0.0

	// New message nodes: 25%
	if evidence.MessageNodeAdded {
		score += 0.25
	}
	totalWeight += 0.25

	// Message content match: 20%
	if evidence.MessageContentMatch {
		score += 0.20
	}
	totalWeight += 0.20

	// Input box clicked and changed: 15%
	if evidence.InputBoxClicked && evidence.InputBoxDiffAfterClick > 0.01 {
		score += 0.15
	}
	totalWeight += 0.15

	// Paste succeeded (input box changed after paste): 10%
	if evidence.InputBoxDiffAfterPaste > 0.01 {
		score += 0.10
	}
	totalWeight += 0.10

	// Chat area change at any timepoint: 15%
	if evidence.ChatAreaDiff300ms > 0.01 || evidence.ChatAreaDiff800ms > 0.01 || evidence.ChatAreaDiff1500ms > 0.01 {
		score += 0.15
	}
	totalWeight += 0.15

	// No failure indicator: 10%
	if !evidence.FailureIndicatorFound {
		score += 0.10
	}
	totalWeight += 0.10

	// Send failure detection penalty: -40% if suspected send failure
	if evidence.SuspectedSendFailed {
		score -= 0.4
		// Ensure score doesn't go below 0
		if score < 0 {
			score = 0
		}
	}
	totalWeight += 0.4 // Add weight for send failure detection

	if totalWeight == 0 {
		return 0
	}
	return score / totalWeight
}

// calculateConfidence calculates confidence score for message verification
func (r *MessageVerificationRules) calculateConfidence(evidence SendVerificationEvidence) float64 {
	score := 0.0
	totalWeight := 0.0

	// New message nodes: 40%
	if evidence.MessageNodeAdded {
		score += 0.4
	}
	totalWeight += 0.4

	// Message content match: 30%
	if evidence.MessageContentMatch {
		score += 0.3
	}
	totalWeight += 0.3

	// Screenshot change: 20%
	if evidence.ScreenshotChanged {
		score += 0.2
	}
	totalWeight += 0.2

	// Chat area diff: 10%
	if evidence.ChatAreaDiff > 0.01 {
		score += 0.1
	}
	totalWeight += 0.1

	// Send failure detection penalty: -30% if suspected send failure
	if evidence.SuspectedSendFailed {
		score -= 0.3
		// Ensure score doesn't go below 0
		if score < 0 {
			score = 0
		}
	}
	totalWeight += 0.3 // Add weight for send failure detection

	if totalWeight == 0 {
		return 0
	}
	return score / totalWeight
}

// ==================== Delivery Assessment Rules ====================

// DeliveryAssessmentRules handles delivery state and confidence determination
type DeliveryAssessmentRules struct{}

// NewDeliveryAssessmentRules creates a new delivery assessment rules instance
func NewDeliveryAssessmentRules() *DeliveryAssessmentRules {
	return &DeliveryAssessmentRules{}
}

// AssessDeliveryState determines delivery state based on focus and message evidence
func (r *DeliveryAssessmentRules) AssessDeliveryState(
	focusEvidence FocusVerificationEvidence,
	messageEvidence SendVerificationEvidence,
) DeliveryAssessment {
	// Calculate overall confidence (focus: 40%, message: 60%)
	// If message evidence is empty (confidence 0), use focus evidence only
	var overallConfidence float64
	var messages []string

	if messageEvidence.Confidence == 0 && messageEvidence.NewMessageNodes == 0 {
		// Focus-only verification
		overallConfidence = focusEvidence.Confidence
		messages = append(messages, fmt.Sprintf("Focus-only verification: confidence=%.2f", overallConfidence))
	} else {
		// Combined verification
		activationWeight := 0.4
		messageWeight := 0.6
		overallConfidence = focusEvidence.Confidence*activationWeight +
			messageEvidence.Confidence*messageWeight
		messages = append(messages, fmt.Sprintf("Combined verification: confidence=%.2f", overallConfidence))
	}

	// Determine delivery state
	var state string
	if overallConfidence >= 0.8 {
		state = "verified"
		messages = append(messages, fmt.Sprintf("Strong verification: confidence=%.2f", overallConfidence))
	} else if overallConfidence >= 0.5 {
		state = "sent_unverified"
		messages = append(messages, fmt.Sprintf("Medium verification: confidence=%.2f", overallConfidence))
	} else {
		state = "unknown"
		messages = append(messages, fmt.Sprintf("Weak verification: confidence=%.2f", overallConfidence))
	}

	return DeliveryAssessment{
		State:      state,
		Confidence: overallConfidence,
		Evidence:   focusEvidence,
		Messages:   messages,
	}

	// Add evidence details to messages
	messages = append(messages, fmt.Sprintf("Focus evidence: %d items, confidence=%.2f",
		focusEvidence.EvidenceCount, focusEvidence.Confidence))
	messages = append(messages, fmt.Sprintf("Message evidence: new_nodes=%d, confidence=%.2f",
		messageEvidence.NewMessageNodes, messageEvidence.Confidence))

	return DeliveryAssessment{
		State:      state,
		Confidence: overallConfidence,
		Evidence:   focusEvidence,
		Messages:   messages,
	}
}

// AssessFocusOnlyState determines delivery state based on focus evidence only
func (r *DeliveryAssessmentRules) AssessFocusOnlyState(
	focusEvidence FocusVerificationEvidence,
) DeliveryAssessment {
	var state string
	var messages []string

	if focusEvidence.Confidence >= 0.8 {
		state = "verified"
		messages = append(messages, fmt.Sprintf("Focus verified: confidence=%.2f", focusEvidence.Confidence))
	} else if focusEvidence.Confidence >= 0.5 {
		state = "sent_unverified"
		messages = append(messages, fmt.Sprintf("Focus unverified: confidence=%.2f", focusEvidence.Confidence))
	} else {
		state = "unknown"
		messages = append(messages, fmt.Sprintf("Focus failed: confidence=%.2f", focusEvidence.Confidence))
	}

	return DeliveryAssessment{
		State:      state,
		Confidence: focusEvidence.Confidence,
		Evidence:   focusEvidence,
		Messages:   messages,
	}
}

// ==================== Utility Functions ====================

// GetWindowWidthFromNodes infers window width from node bounds
func GetWindowWidthFromNodes(flatNodes []windows.AccessibleNode) int {
	if len(flatNodes) > 0 && len(flatNodes[0].Bounds) == 4 {
		node := flatNodes[0]
		windowWidth := node.Bounds[0] + node.Bounds[2]
		if windowWidth >= 400 {
			return windowWidth
		}
	}
	return 800 // default width
}

// ConvertFocusEvidenceToDiagnostics converts focus evidence to diagnostic context
func ConvertFocusEvidenceToDiagnostics(evidence FocusVerificationEvidence) map[string]string {
	return map[string]string{
		"locate_source":          evidence.LocateSource,
		"node_still_exists":      strconv.FormatBool(evidence.NodeStillExists),
		"node_has_active_state":  strconv.FormatBool(evidence.NodeHasActiveState),
		"title_contains_target":  strconv.FormatBool(evidence.TitleContainsTarget),
		"panel_switch_detected":  strconv.FormatBool(evidence.PanelSwitchDetected),
		"message_area_visible":   strconv.FormatBool(evidence.MessageAreaVisible),
		"evidence_count":         strconv.Itoa(evidence.EvidenceCount),
		"confidence":             fmt.Sprintf("%.2f", evidence.Confidence),
	}
}

// ConvertMessageEvidenceToDiagnostics converts message evidence to diagnostic context
func ConvertMessageEvidenceToDiagnostics(evidence SendVerificationEvidence) map[string]string {
	return map[string]string{
		"new_message_nodes":          strconv.Itoa(evidence.NewMessageNodes),
		"message_node_added":         strconv.FormatBool(evidence.MessageNodeAdded),
		"message_content_match":      strconv.FormatBool(evidence.MessageContentMatch),
		"screenshot_changed":         strconv.FormatBool(evidence.ScreenshotChanged),
		"chat_area_diff":             fmt.Sprintf("%.2f", evidence.ChatAreaDiff),
		"input_box_clicked":          strconv.FormatBool(evidence.InputBoxClicked),
		"input_box_diff_after_click": fmt.Sprintf("%.3f", evidence.InputBoxDiffAfterClick),
		"input_box_diff_after_paste": fmt.Sprintf("%.3f", evidence.InputBoxDiffAfterPaste),
		"suspected_input_not_focused": strconv.FormatBool(evidence.SuspectedInputNotFocused),
		"suspected_paste_failed":     strconv.FormatBool(evidence.SuspectedPasteFailed),
		"chat_area_diff_300ms":       fmt.Sprintf("%.3f", evidence.ChatAreaDiff300ms),
		"chat_area_diff_800ms":       fmt.Sprintf("%.3f", evidence.ChatAreaDiff800ms),
		"chat_area_diff_1500ms":      fmt.Sprintf("%.3f", evidence.ChatAreaDiff1500ms),
		"failure_indicator_found":    strconv.FormatBool(evidence.FailureIndicatorFound),
		"failure_indicator_bbox":     fmt.Sprintf("%d_%d_%d_%d", evidence.FailureIndicatorBbox[0], evidence.FailureIndicatorBbox[1], evidence.FailureIndicatorBbox[2], evidence.FailureIndicatorBbox[3]),
		"suspected_send_failed":      strconv.FormatBool(evidence.SuspectedSendFailed),
		"send_failure_reason":        evidence.SendFailureReason,
		"failure_signal_source":      evidence.FailureSignalSource,
		"send_verification_quality":  "enhanced_differential", // Updated to reflect enhanced verification
		"confidence":                 fmt.Sprintf("%.2f", evidence.Confidence),
	}
}

// ConvertDeliveryAssessmentToDiagnostics converts delivery assessment to diagnostic context
func ConvertDeliveryAssessmentToDiagnostics(assessment DeliveryAssessment) map[string]string {
	return map[string]string{
		"delivery_state": assessment.State,
		"confidence":     fmt.Sprintf("%.2f", assessment.Confidence),
		"messages":       strings.Join(assessment.Messages, "; "),
	}
}
