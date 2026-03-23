package wechat

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/yourorg/auto-customer-service/internal/agent/windows"
	"github.com/yourorg/auto-customer-service/pkg/protocol"
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
	// Rule 1: Check role - must be list item
	if node.Role != "list item" && node.Role != "ListItem" {
		return false
	}

	// Rule 2: Check name is non-empty
	if node.Name == "" {
		return false
	}

	// Rule 3: Check bounds are valid
	if len(node.Bounds) != 4 {
		return false
	}
	bounds := node.Bounds
	// Check for negative or zero dimensions
	if bounds[0] < 0 || bounds[1] < 0 || bounds[2] <= 0 || bounds[3] <= 0 {
		return false
	}

	// Rule 4: Check position - must be in left 1/3 of window (contact list area)
	listAreaThreshold := windowWidth / 3
	if bounds[0] > listAreaThreshold {
		return false
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

	// Calculate confidence
	evidence.Confidence = r.calculateConfidence(evidence)

	return evidence
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
	activationWeight := 0.4
	messageWeight := 0.6

	overallConfidence := focusEvidence.Confidence*activationWeight +
		messageEvidence.Confidence*messageWeight

	// Determine delivery state
	var state string
	var messages []string

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
		"new_message_nodes":      strconv.Itoa(evidence.NewMessageNodes),
		"message_node_added":     strconv.FormatBool(evidence.MessageNodeAdded),
		"message_content_match":  strconv.FormatBool(evidence.MessageContentMatch),
		"screenshot_changed":     strconv.FormatBool(evidence.ScreenshotChanged),
		"chat_area_diff":         fmt.Sprintf("%.2f", evidence.ChatAreaDiff),
		"confidence":             fmt.Sprintf("%.2f", evidence.Confidence),
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
