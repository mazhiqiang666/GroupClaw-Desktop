package wechat

import (
	"fmt"
	"strings"

	"github.com/yourorg/auto-customer-service/internal/agent/windows"
	"github.com/yourorg/auto-customer-service/pkg/protocol"
)

// VerificationEngine handles all verification logic for WeChat adapter
type VerificationEngine struct{}

// NewVerificationEngine creates a new verification engine
func NewVerificationEngine() *VerificationEngine {
	return &VerificationEngine{}
}

// ActivationEvidence represents evidence for session activation
type ActivationEvidence struct {
	HasActiveState    bool
	HasTitleChange    bool
	HasPanelSwitch    bool
	NodeStillExists   bool
	LocateSource      string
	Confidence        float64
}

// MessageEvidence represents evidence for message sending
type MessageEvidence struct {
	NewMessageNodes   int
	NewMessageText    []string
	ScreenshotChanged bool
	ChatAreaDiff      float64
	Confidence        float64
}

// PathSystem handles node path generation and parsing
type PathSystem struct{}

// NewPathSystem creates a new path system
func NewPathSystem() *PathSystem {
	return &PathSystem{}
}

// GeneratePath generates a hierarchical path for a node
// Format: [0].[3].[2] for node at index 2 under node at index 3 under root node 0
func (ps *PathSystem) GeneratePath(node windows.AccessibleNode, parentPath string, index int) string {
	if parentPath == "" {
		return fmt.Sprintf("[%d]", index)
	}
	return fmt.Sprintf("%s.[%d]", parentPath, index)
}

// ParsePath parses a hierarchical path string into indices
// Example: "[0].[3].[2]" -> [0, 3, 2]
func (ps *PathSystem) ParsePath(path string) ([]int, error) {
	if path == "" {
		return nil, fmt.Errorf("empty path")
	}

	// Remove outer brackets if present
	path = strings.Trim(path, "[]")
	parts := strings.Split(path, "].[")

	indices := make([]int, len(parts))
	for i, part := range parts {
		// Clean up any remaining brackets
		part = strings.Trim(part, "[]")
		idx, err := fmt.Sscanf(part, "%d", &indices[i])
		if err != nil || idx != 1 {
			return nil, fmt.Errorf("invalid path format: %s", path)
		}
	}

	return indices, nil
}

// FindNodeByPath finds a node using hierarchical path
func (ps *PathSystem) FindNodeByPath(nodes []windows.AccessibleNode, path string) (*windows.AccessibleNode, error) {
	indices, err := ps.ParsePath(path)
	if err != nil {
		return nil, err
	}

	currentNodes := nodes
	for i, idx := range indices {
		if idx < 0 || idx >= len(currentNodes) {
			return nil, fmt.Errorf("index %d out of range at level %d", idx, i)
		}

		node := currentNodes[idx]
		if i == len(indices)-1 {
			// Last index, return the node
			return &node, nil
		}
		// Continue to next level
		currentNodes = node.Children
	}

	return nil, fmt.Errorf("path not found: %s", path)
}

// FlattenNodesWithPath flattens node tree and generates paths
func (ps *PathSystem) FlattenNodesWithPath(
	nodes []windows.AccessibleNode,
	parentPath string,
	depth int,
	maxDepth int,
) []windows.AccessibleNode {
	if depth >= maxDepth {
		return nodes
	}

	result := make([]windows.AccessibleNode, 0, len(nodes))
	for i, node := range nodes {
		// Generate path for this node
		nodePath := ps.GeneratePath(node, parentPath, i)
		node.TreePath = nodePath

		result = append(result, node)
		if len(node.Children) > 0 {
			childNodes := ps.FlattenNodesWithPath(node.Children, nodePath, depth+1, maxDepth)
			result = append(result, childNodes...)
		}
	}
	return result
}

// EvidenceCollector collects and scores verification evidence
type EvidenceCollector struct{}

// NewEvidenceCollector creates a new evidence collector
func NewEvidenceCollector() *EvidenceCollector {
	return &EvidenceCollector{}
}

// CollectActivationEvidence collects evidence for session activation
func (ec *EvidenceCollector) CollectActivationEvidence(
	conv protocol.ConversationRef,
	nodes []windows.AccessibleNode,
	originalNodes []windows.AccessibleNode,
	locateSource string,
) ActivationEvidence {
	evidence := ActivationEvidence{
		LocateSource: locateSource,
	}

	// Check 1: Node still exists (basic check)
	for _, node := range nodes {
		if node.Name == conv.DisplayName {
			evidence.NodeStillExists = true
			break
		}
	}

	// Check 2: Active state detection (role contains "selected" or "active")
	for _, node := range nodes {
		role := strings.ToLower(node.Role)
		if strings.Contains(role, "selected") || strings.Contains(role, "active") {
			evidence.HasActiveState = true
			break
		}
	}

	// Check 3: Title change detection
	// Look for a node with the target conversation name in a prominent position
	for _, node := range nodes {
		if node.Name == conv.DisplayName && node.Bounds[0] < 200 {
			evidence.HasTitleChange = true
			break
		}
	}

	// Check 4: Panel switch detection
	// Compare node counts between original and current
	if len(originalNodes) > 0 && len(nodes) > 0 {
		// If node count changed significantly, panel likely switched
		ratio := float64(len(nodes)) / float64(len(originalNodes))
		if ratio < 0.8 || ratio > 1.2 {
			evidence.HasPanelSwitch = true
		}
	}

	// Calculate confidence based on evidence
	evidence.Confidence = ec.scoreActivationEvidence(evidence)

	return evidence
}

// scoreActivationEvidence calculates confidence score for activation evidence
func (ec *EvidenceCollector) scoreActivationEvidence(evidence ActivationEvidence) float64 {
	score := 0.0
	totalWeight := 0.0

	// Node existence: 20%
	if evidence.NodeStillExists {
		score += 0.2
	}
	totalWeight += 0.2

	// Active state: 30%
	if evidence.HasActiveState {
		score += 0.3
	}
	totalWeight += 0.3

	// Title change: 30%
	if evidence.HasTitleChange {
		score += 0.3
	}
	totalWeight += 0.3

	// Panel switch: 20%
	if evidence.HasPanelSwitch {
		score += 0.2
	}
	totalWeight += 0.2

	if totalWeight == 0 {
		return 0
	}
	return score / totalWeight
}

// CollectMessageEvidence collects evidence for message sending
func (ec *EvidenceCollector) CollectMessageEvidence(
	beforeNodes []windows.AccessibleNode,
	afterNodes []windows.AccessibleNode,
	beforeScreenshot []byte,
	afterScreenshot []byte,
	chatAreaBounds [4]int,
) MessageEvidence {
	evidence := MessageEvidence{}

	// Find new message nodes
	newNodes := ec.findNewMessageNodes(beforeNodes, afterNodes)
	evidence.NewMessageNodes = len(newNodes)

	// Extract text from new nodes
	for _, node := range newNodes {
		if node.Name != "" {
			evidence.NewMessageText = append(evidence.NewMessageText, node.Name)
		}
	}

	// Check screenshot change in chat area
	if len(beforeScreenshot) > 0 && len(afterScreenshot) > 0 {
		evidence.ScreenshotChanged = ec.checkScreenshotChange(beforeScreenshot, afterScreenshot, chatAreaBounds)
	}

	// Calculate chat area diff
	evidence.ChatAreaDiff = ec.CalculateChatAreaDiff(beforeScreenshot, afterScreenshot, chatAreaBounds)

	// Calculate confidence
	evidence.Confidence = ec.scoreMessageEvidence(evidence)

	return evidence
}

// findNewMessageNodes finds nodes that appear after sending
func (ec *EvidenceCollector) findNewMessageNodes(
	before []windows.AccessibleNode,
	after []windows.AccessibleNode,
) []windows.AccessibleNode {
	beforeMap := make(map[string]bool)
	for _, node := range before {
		key := ec.nodeKey(node)
		beforeMap[key] = true
	}

	var newNodes []windows.AccessibleNode
	for _, node := range after {
		key := ec.nodeKey(node)
		if !beforeMap[key] {
			newNodes = append(newNodes, node)
		}
	}
	return newNodes
}

// nodeKey generates a unique key for a node
func (ec *EvidenceCollector) nodeKey(node windows.AccessibleNode) string {
	return fmt.Sprintf("%s|%s|%v", node.Name, node.Role, node.Bounds)
}

// checkScreenshotChange checks if screenshot changed in chat area
func (ec *EvidenceCollector) checkScreenshotChange(
	before []byte,
	after []byte,
	chatAreaBounds [4]int,
) bool {
	if len(before) != len(after) {
		return true
	}

	// Simple byte comparison (in real implementation, would compare specific region)
	for i := 0; i < len(before) && i < len(after); i++ {
		if before[i] != after[i] {
			return true
		}
	}
	return false
}

// CalculateChatAreaDiff calculates difference percentage in chat area
func (ec *EvidenceCollector) CalculateChatAreaDiff(
	before []byte,
	after []byte,
	chatAreaBounds [4]int,
) float64 {
	if len(before) == 0 || len(after) == 0 {
		return 0
	}

	// In real implementation, would extract and compare chat area region
	// For now, return simple comparison
	sameBytes := 0
	totalBytes := len(before)
	if len(after) < totalBytes {
		totalBytes = len(after)
	}

	for i := 0; i < totalBytes; i++ {
		if before[i] == after[i] {
			sameBytes++
		}
	}

	return 1.0 - float64(sameBytes)/float64(totalBytes)
}

// scoreMessageEvidence calculates confidence score for message evidence
func (ec *EvidenceCollector) scoreMessageEvidence(evidence MessageEvidence) float64 {
	score := 0.0
	totalWeight := 0.0

	// New message nodes: 40%
	if evidence.NewMessageNodes > 0 {
		score += 0.4
	}
	totalWeight += 0.4

	// New message text: 30%
	if len(evidence.NewMessageText) > 0 {
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

// DeliveryState determines the delivery state based on evidence
func (ec *EvidenceCollector) DetermineDeliveryState(
	activationEvidence ActivationEvidence,
	messageEvidence MessageEvidence,
) (deliveryState string, confidence float64) {
	// Calculate overall confidence
	activationWeight := 0.4
	messageWeight := 0.6

	overallConfidence := activationEvidence.Confidence*activationWeight +
		messageEvidence.Confidence*messageWeight

	// Determine delivery state
	if overallConfidence >= 0.8 {
		deliveryState = "verified"
	} else if overallConfidence >= 0.5 {
		deliveryState = "sent_unverified"
	} else {
		deliveryState = "unknown"
	}

	return deliveryState, overallConfidence
}

// MessageClassifier classifies node types for message detection
type MessageClassifier struct{}

// NewMessageClassifier creates a new message classifier
func NewMessageClassifier() *MessageClassifier {
	return &MessageClassifier{}
}

// NodeType represents different types of nodes
type NodeType int

const (
	NodeTypeUnknown NodeType = iota
	NodeTypeMessageBubble
	NodeTypeInputBox
	NodeTypeTitle
	NodeTypeSystemPrompt
	NodeTypeNormalText
)

// ClassifyNode classifies a node by its type
func (mc *MessageClassifier) ClassifyNode(node windows.AccessibleNode) NodeType {
	role := strings.ToLower(node.Role)
	bounds := node.Bounds

	// Check for input box (edit control)
	if strings.Contains(role, "edit") || strings.Contains(role, "textbox") {
		return NodeTypeInputBox
	}

	// Check for title (usually at top, has specific bounds)
	if bounds[1] < 50 && bounds[3] < 30 {
		return NodeTypeTitle
	}

	// Check for system prompt (often has specific role or bounds)
	if strings.Contains(role, "alert") || strings.Contains(role, "status") {
		return NodeTypeSystemPrompt
	}

	// Check for message bubble (right-aligned, has reasonable height)
	if bounds[0] > 100 && bounds[3] > 20 && bounds[3] < 100 {
		return NodeTypeMessageBubble
	}

	// Check for normal text
	if strings.Contains(role, "text") || strings.Contains(role, "static") {
		return NodeTypeNormalText
	}

	return NodeTypeUnknown
}

// IsMessageCandidate checks if a node is a candidate for message content
func (mc *MessageClassifier) IsMessageCandidate(node windows.AccessibleNode) bool {
	nodeType := mc.ClassifyNode(node)
	return nodeType == NodeTypeMessageBubble || nodeType == NodeTypeNormalText
}

// FilterMessageAreaNodes filters nodes to only include message area nodes
func (mc *MessageClassifier) FilterMessageAreaNodes(
	nodes []windows.AccessibleNode,
	windowHandle uintptr,
) []windows.AccessibleNode {
	var result []windows.AccessibleNode

	for _, node := range nodes {
		// Skip nodes outside reasonable bounds
		if node.Bounds[2] <= 0 || node.Bounds[3] <= 0 {
			continue
		}

		// Skip input box nodes
		if mc.ClassifyNode(node) == NodeTypeInputBox {
			continue
		}

		// Skip title nodes (top area)
		if node.Bounds[1] < 50 {
			continue
		}

		// Only include text/static nodes with reasonable content
		if node.Name != "" && len(node.Name) > 0 {
			result = append(result, node)
		}
	}

	return result
}
