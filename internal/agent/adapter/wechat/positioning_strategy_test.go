package wechat

import (
	"testing"

	"github.com/yourorg/auto-customer-service/internal/agent/windows"
	"github.com/yourorg/auto-customer-service/pkg/protocol"
)

// ==================== PositioningStrategyRules Basic Tests ====================

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

	// Test strategy 3: Stable key match (PreviewText)
	// Generate the expected stable key for the node
	expectedKey := "[1]||list item|李四|50_150_150_40"
	convStableKey := protocol.ConversationRef{
		DisplayName: "李四",
		PreviewText: expectedKey,
	}

	resultStableKey := rules.FindNodeByStrategy(nodes, convStableKey)
	if resultStableKey.Node == nil || resultStableKey.Node.Name != "李四" {
		t.Errorf("Expected to find '李四' by stable key match, got %v", resultStableKey.Node)
	}
	if resultStableKey.Source != "stable_key" {
		t.Errorf("Expected source 'stable_key', got %s", resultStableKey.Source)
	}

	// Test strategy 4: Name match only
	convNameMatch := protocol.ConversationRef{
		DisplayName: "张三",
	}

	resultNameMatch := rules.FindNodeByStrategy(nodes, convNameMatch)
	if resultNameMatch.Node == nil || resultNameMatch.Node.Name != "张三" {
		t.Errorf("Expected to find '张三' by name match, got %v", resultNameMatch.Node)
	}
	if resultNameMatch.Source != "name_match" {
		t.Errorf("Expected source 'name_match', got %s", resultNameMatch.Source)
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

// ==================== PositioningStrategyRules Dirty Data Tests ====================

func TestPositioningStrategyRules_DirtyData_FindNodeWithInvalidHints(t *testing.T) {
	pathSystem := NewPathSystem()
	rules := NewPositioningStrategyRules(pathSystem)

	nodes := []windows.AccessibleNode{
		{Name: "张三", Role: "list item", Bounds: [4]int{50, 100, 150, 40}, TreePath: "[0]"},
		{Name: "李四", Role: "list item", Bounds: [4]int{50, 150, 150, 40}, TreePath: "[1]"},
	}

	tests := []struct {
		name        string
		conv        protocol.ConversationRef
		expectFound bool
		expectName  string
	}{
		{
			name: "Empty hints",
			conv: protocol.ConversationRef{
				DisplayName:         "张三",
				ListNeighborhoodHint: []string{},
			},
			expectFound: true,
			expectName:  "张三",
		},
		{
			name: "Invalid tree path format",
			conv: protocol.ConversationRef{
				DisplayName:         "张三",
				ListNeighborhoodHint: []string{"invalid", "bounds:50_100_150_40"},
			},
			expectFound: true,
			expectName:  "张三",
		},
		{
			name: "Non-matching tree path",
			conv: protocol.ConversationRef{
				DisplayName:         "张三",
				ListNeighborhoodHint: []string{"[99]", "bounds:50_100_150_40"},
			},
			expectFound: true,
			expectName:  "张三",
		},
		{
			name: "Invalid bounds format",
			conv: protocol.ConversationRef{
				DisplayName:         "张三",
				ListNeighborhoodHint: []string{"[0]", "invalid_bounds"},
			},
			expectFound: true,
			expectName:  "张三",
		},
		{
			name: "Mismatched bounds",
			conv: protocol.ConversationRef{
				DisplayName:         "张三",
				ListNeighborhoodHint: []string{"[99]", "bounds:999_999_999_999"},
			},
			expectFound: true,
			expectName:  "张三",
		},
		{
			name: "Non-existent contact",
			conv: protocol.ConversationRef{
				DisplayName: "不存在的人",
			},
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rules.FindNodeByStrategy(nodes, tt.conv)

			if tt.expectFound {
				if result.Node == nil {
					t.Errorf("Expected to find node, got nil")
				} else if result.Node.Name != tt.expectName {
					t.Errorf("Expected name '%s', got '%s'", tt.expectName, result.Node.Name)
				}
			} else {
				if result.Node != nil {
					t.Errorf("Expected no match, got node with name '%s'", result.Node.Name)
				}
				if result.Source != "not_found" {
					t.Errorf("Expected source 'not_found', got '%s'", result.Source)
				}
			}
		})
	}
}

func TestPositioningStrategyRules_DirtyData_CalculateClickPosition_InvalidNodes(t *testing.T) {
	pathSystem := NewPathSystem()
	rules := NewPositioningStrategyRules(pathSystem)

	tests := []struct {
		name    string
		node    *windows.AccessibleNode
		wantOk  bool
		wantX   int
		wantY   int
	}{
		{
			name:   "Nil node",
			node:   nil,
			wantOk: false,
		},
		{
			name:   "Empty bounds",
			node:   &windows.AccessibleNode{Name: "Test", Bounds: [4]int{}},
			wantOk: false,
		},
		{
			name:   "Partial bounds",
			node:   &windows.AccessibleNode{Name: "Test", Bounds: [4]int{100}},
			wantOk: false,
		},
		{
			name:   "Zero width and height",
			node:   &windows.AccessibleNode{Name: "Test", Bounds: [4]int{100, 200, 0, 0}},
			wantOk: false, // Zero dimensions are invalid for clicking
		},
		{
			name:   "Negative bounds",
			node:   &windows.AccessibleNode{Name: "Test", Bounds: [4]int{-100, -200, 300, 400}},
			wantOk: true, // Negative position is allowed, positive dimensions are valid
			wantX:  50,
			wantY:  0,
		},
		{
			name:   "Very large bounds",
			node:   &windows.AccessibleNode{Name: "Test", Bounds: [4]int{10000, 20000, 5000, 3000}},
			wantOk: true,
			wantX:  12500,
			wantY:  21500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x, y, ok := rules.CalculateClickPosition(tt.node)
			if ok != tt.wantOk {
				t.Errorf("CalculateClickPosition() ok = %v, want %v", ok, tt.wantOk)
			}
			if ok && tt.wantOk {
				if x != tt.wantX {
					t.Errorf("CalculateClickPosition() x = %d, want %d", x, tt.wantX)
				}
				if y != tt.wantY {
					t.Errorf("CalculateClickPosition() y = %d, want %d", y, tt.wantY)
				}
			}
		})
	}
}

// ==================== PositioningStrategyRules Complex Scenarios ====================

func TestPositioningStrategyRules_ComplexScenarios(t *testing.T) {
	pathSystem := NewPathSystem()
	rules := NewPositioningStrategyRules(pathSystem)

	// Complex tree structure with same-name contacts
	nodes := []windows.AccessibleNode{
		{Name: "张三", Role: "list item", Bounds: [4]int{50, 100, 150, 40}, TreePath: "[0]"},
		{Name: "张三", Role: "list item", Bounds: [4]int{50, 150, 150, 40}, TreePath: "[1]"},
		{Name: "李四", Role: "list item", Bounds: [4]int{50, 200, 150, 40}, TreePath: "[2]"},
		{Name: "王五", Role: "list item", Bounds: [4]int{50, 250, 150, 40}, TreePath: "[3]"},
	}

	tests := []struct {
		name        string
		conv        protocol.ConversationRef
		expectFound bool
		expectName  string
		expectSrc   string
	}{
		{
			name: "Tree path + name match (first 张三)",
			conv: protocol.ConversationRef{
				DisplayName:         "张三",
				ListNeighborhoodHint: []string{"[0]", "bounds:50_100_150_40"},
			},
			expectFound: true,
			expectName:  "张三",
			expectSrc:   "tree_path_name",
		},
		{
			name: "Tree path + name match (second 张三)",
			conv: protocol.ConversationRef{
				DisplayName:         "张三",
				ListNeighborhoodHint: []string{"[1]", "bounds:50_150_150_40"},
			},
			expectFound: true,
			expectName:  "张三",
			expectSrc:   "tree_path_name",
		},
		{
			name: "Bounds match fallback",
			conv: protocol.ConversationRef{
				DisplayName:         "李四",
				ListNeighborhoodHint: []string{"[99]", "bounds:50_200_150_40"},
			},
			expectFound: true,
			expectName:  "李四",
			expectSrc:   "bounds_match",
		},
		{
			name: "Name match only fallback",
			conv: protocol.ConversationRef{
				DisplayName: "王五",
			},
			expectFound: true,
			expectName:  "王五",
			expectSrc:   "name_match",
		},
		{
			name: "No match - different name",
			conv: protocol.ConversationRef{
				DisplayName: "不存在的人",
			},
			expectFound: false,
			expectSrc:   "not_found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rules.FindNodeByStrategy(nodes, tt.conv)

			if tt.expectFound {
				if result.Node == nil {
					t.Errorf("Expected to find node, got nil")
				} else if result.Node.Name != tt.expectName {
					t.Errorf("Expected name '%s', got '%s'", tt.expectName, result.Node.Name)
				}
			} else {
				if result.Node != nil {
					t.Errorf("Expected no match, got node with name '%s'", result.Node.Name)
				}
			}

			if result.Source != tt.expectSrc {
				t.Errorf("Expected source '%s', got '%s'", tt.expectSrc, result.Source)
			}
		})
	}
}
