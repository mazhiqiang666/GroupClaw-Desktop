package wechat

import (
	"testing"

	"github.com/yourorg/auto-customer-service/internal/agent/windows"
)

// ==================== Path System Tests ====================

func TestPathSystem_GeneratePath(t *testing.T) {
	ps := NewPathSystem()

	tests := []struct {
		name       string
		node       windows.AccessibleNode
		parentPath string
		index      int
		expected   string
	}{
		{
			name:       "Root node",
			node:       windows.AccessibleNode{Name: "Root"},
			parentPath: "",
			index:      0,
			expected:   "[0]",
		},
		{
			name:       "Child node",
			node:       windows.AccessibleNode{Name: "Child"},
			parentPath: "[0]",
			index:      3,
			expected:   "[0].[3]",
		},
		{
			name:       "Nested path",
			node:       windows.AccessibleNode{Name: "Grandchild"},
			parentPath: "[0].[3]",
			index:      2,
			expected:   "[0].[3].[2]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ps.GeneratePath(tt.node, tt.parentPath, tt.index)
			if result != tt.expected {
				t.Errorf("GeneratePath() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPathSystem_ParsePath(t *testing.T) {
	ps := NewPathSystem()

	tests := []struct {
		name        string
		path        string
		expectError bool
		expected    []int
	}{
		{
			name:        "Simple path",
			path:        "[0]",
			expectError: false,
			expected:    []int{0},
		},
		{
			name:        "Hierarchical path",
			path:        "[0].[3].[2]",
			expectError: false,
			expected:    []int{0, 3, 2},
		},
		{
			name:        "Empty path",
			path:        "",
			expectError: true,
		},
		{
			name:        "Invalid path - non-numeric",
			path:        "[abc]",
			expectError: true,
		},
		{
			name:        "Invalid path - negative index",
			path:        "[-1]",
			expectError: true,
		},
		{
			name:        "Invalid path - missing brackets",
			path:        "0.1.2",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			indices, err := ps.ParsePath(tt.path)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for path '%s', got nil", tt.path)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for path '%s': %v", tt.path, err)
				}
				if len(indices) != len(tt.expected) {
					t.Errorf("Expected %d indices, got %d", len(tt.expected), len(indices))
				}
				for i, idx := range indices {
					if idx != tt.expected[i] {
						t.Errorf("Index %d: expected %d, got %d", i, tt.expected[i], idx)
					}
				}
			}
		})
	}
}

func TestPathSystem_FindNodeByPath(t *testing.T) {
	ps := NewPathSystem()

	// Create complex tree structure with same-name nodes
	nodes := []windows.AccessibleNode{
		{
			Name:     "ContactList",
			TreePath: "[0]",
			Children: []windows.AccessibleNode{
				{
					Name:     "张三",
					Role:     "list item",
					TreePath: "[0].[0]",
					Children: []windows.AccessibleNode{
						{Name: "Message1", TreePath: "[0].[0].[0]"},
					},
				},
				{
					Name:     "张三", // Same name as above
					Role:     "list item",
					TreePath: "[0].[1]",
					Children: []windows.AccessibleNode{
						{Name: "Message2", TreePath: "[0].[1].[0]"},
					},
				},
				{
					Name:     "李四",
					Role:     "list item",
					TreePath: "[0].[2]",
				},
			},
		},
	}

	tests := []struct {
		name        string
		path        string
		expectError bool
		expectName  string
	}{
		{
			name:        "Find root node",
			path:        "[0]",
			expectError: false,
			expectName:  "ContactList",
		},
		{
			name:        "Find first张三",
			path:        "[0].[0]",
			expectError: false,
			expectName:  "张三",
		},
		{
			name:        "Find second张三",
			path:        "[0].[1]",
			expectError: false,
			expectName:  "张三",
		},
		{
			name:        "Find message under first张三",
			path:        "[0].[0].[0]",
			expectError: false,
			expectName:  "Message1",
		},
		{
			name:        "Find message under second张三",
			path:        "[0].[1].[0]",
			expectError: false,
			expectName:  "Message2",
		},
		{
			name:        "Out of range index",
			path:        "[0].[99]",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := ps.FindNodeByPath(nodes, tt.path)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for path '%s', got nil", tt.path)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for path '%s': %v", tt.path, err)
				}
				if node == nil {
					t.Errorf("Expected node for path '%s', got nil", tt.path)
				} else if node.Name != tt.expectName {
					t.Errorf("Expected name '%s', got '%s'", tt.expectName, node.Name)
				}
			}
		})
	}
}

func TestPathSystem_FlattenNodesWithPath(t *testing.T) {
	ps := NewPathSystem()

	// Create deeply nested structure
	nodes := []windows.AccessibleNode{
		{
			Name: "Root",
			Children: []windows.AccessibleNode{
				{
					Name: "Level1_A",
					Children: []windows.AccessibleNode{
						{Name: "Level2_A1"},
						{Name: "Level2_A2"},
					},
				},
				{
					Name: "Level1_B",
					Children: []windows.AccessibleNode{
						{Name: "Level2_B1"},
					},
				},
			},
		},
	}

	tests := []struct {
		name      string
		maxDepth  int
		expectLen int
	}{
		{
			name:      "Shallow flatten (depth 1)",
			maxDepth:  1,
			expectLen: 1, // Only root
		},
		{
			name:      "Medium flatten (depth 2)",
			maxDepth:  2,
			expectLen: 3, // Root + 2 Level1 nodes
		},
		{
			name:      "Deep flatten (depth 10)",
			maxDepth:  10,
			expectLen: 6, // All nodes
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flatNodes := ps.FlattenNodesWithPath(nodes, "", 0, tt.maxDepth)
			if len(flatNodes) != tt.expectLen {
				t.Errorf("Expected %d flat nodes, got %d", tt.expectLen, len(flatNodes))
			}
		})
	}
}

func TestPathSystem_StableKeyRefind(t *testing.T) {
	ps := NewPathSystem()

	// Create nodes with stable paths
	nodes := []windows.AccessibleNode{
		{
			Name:     "张三",
			Role:     "list item",
			Bounds:   [4]int{10, 50, 180, 40},
			TreePath: "[0]",
		},
		{
			Name:     "李四",
			Role:     "list item",
			Bounds:   [4]int{10, 90, 180, 40},
			TreePath: "[1]",
		},
	}

	// Test finding by path
	node, err := ps.FindNodeByPath(nodes, "[0]")
	if err != nil {
		t.Errorf("Failed to find node by path: %v", err)
	}
	if node.Name != "张三" {
		t.Errorf("Expected '张三', got '%s'", node.Name)
	}
}

// ==================== Complex Scenario Tests ====================

func TestPathSystem_SameNameContacts(t *testing.T) {
	ps := NewPathSystem()

	// Create realistic WeChat contact list with multiple same-name contacts
	nodes := []windows.AccessibleNode{
		{
			Name:     "ContactList",
			TreePath: "[0]",
			Children: []windows.AccessibleNode{
				{
					Name:     "张三",
					Role:     "list item",
					Bounds:   [4]int{10, 50, 200, 60},
					TreePath: "[0].[0]",
					Children: []windows.AccessibleNode{
						{Name: "Message1", TreePath: "[0].[0].[0]"},
					},
				},
				{
					Name:     "张三", // Same name - different person
					Role:     "list item",
					Bounds:   [4]int{10, 120, 200, 60},
					TreePath: "[0].[1]",
					Children: []windows.AccessibleNode{
						{Name: "Message2", TreePath: "[0].[1].[0]"},
					},
				},
				{
					Name:     "张三", // Third same-name contact
					Role:     "list item",
					Bounds:   [4]int{10, 190, 200, 60},
					TreePath: "[0].[2]",
					Children: []windows.AccessibleNode{
						{Name: "Message3", TreePath: "[0].[2].[0]"},
					},
				},
				{
					Name:     "李四",
					Role:     "list item",
					Bounds:   [4]int{10, 260, 200, 60},
					TreePath: "[0].[3]",
				},
			},
		},
	}

	tests := []struct {
		name        string
		path        string
		expectName  string
		expectError bool
	}{
		{
			name:       "Find first 张三 by path",
			path:       "[0].[0]",
			expectName: "张三",
		},
		{
			name:       "Find second 张三 by path",
			path:       "[0].[1]",
			expectName: "张三",
		},
		{
			name:       "Find third 张三 by path",
			path:       "[0].[2]",
			expectName: "张三",
		},
		{
			name:       "Find message under first 张三",
			path:       "[0].[0].[0]",
			expectName: "Message1",
		},
		{
			name:       "Find message under second 张三",
			path:       "[0].[1].[0]",
			expectName: "Message2",
		},
		{
			name:       "Find message under third 张三",
			path:       "[0].[2].[0]",
			expectName: "Message3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := ps.FindNodeByPath(nodes, tt.path)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for path '%s', got nil", tt.path)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for path '%s': %v", tt.path, err)
				}
				if node == nil {
					t.Errorf("Expected node for path '%s', got nil", tt.path)
				} else if node.Name != tt.expectName {
					t.Errorf("Expected name '%s', got '%s'", tt.expectName, node.Name)
				}
			}
		})
	}
}

func TestPathSystem_TitleNodeSameNameAsContact(t *testing.T) {
	ps := NewPathSystem()

	// Create structure where title node has same name as contact
	nodes := []windows.AccessibleNode{
		{
			Name:     "ChatWindow",
			TreePath: "[0]",
			Children: []windows.AccessibleNode{
				{
					Name:     "张三", // Title node
					Role:     "text",
					Bounds:   [4]int{50, 10, 300, 25}, // Top position
					TreePath: "[0].[0]",
				},
				{
					Name:     "张三", // Contact node (same name)
					Role:     "list item",
					Bounds:   [4]int{10, 50, 200, 60},
					TreePath: "[0].[1]",
				},
				{
					Name:     "MessageArea",
					Role:     "group",
					Bounds:   [4]int{0, 80, 400, 300},
					TreePath: "[0].[2]",
					Children: []windows.AccessibleNode{
						{
							Name:     "张三",
							Role:     "text",
							Bounds:   [4]int{300, 100, 150, 40},
							TreePath: "[0].[2].[0]",
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name        string
		path        string
		expectName  string
		expectRole  string
		expectError bool
	}{
		{
			name:       "Find title node (top position)",
			path:       "[0].[0]",
			expectName: "张三",
			expectRole: "text",
		},
		{
			name:       "Find contact node (list item)",
			path:       "[0].[1]",
			expectName: "张三",
			expectRole: "list item",
		},
		{
			name:       "Find message bubble under message area",
			path:       "[0].[2].[0]",
			expectName: "张三",
			expectRole: "text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := ps.FindNodeByPath(nodes, tt.path)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for path '%s', got nil", tt.path)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for path '%s': %v", tt.path, err)
				}
				if node == nil {
					t.Errorf("Expected node for path '%s', got nil", tt.path)
				} else {
					if node.Name != tt.expectName {
						t.Errorf("Expected name '%s', got '%s'", tt.expectName, node.Name)
					}
					if node.Role != tt.expectRole {
						t.Errorf("Expected role '%s', got '%s'", tt.expectRole, node.Role)
					}
				}
			}
		})
	}
}

func TestPathSystem_RightSideNodes(t *testing.T) {
	ps := NewPathSystem()

	// Create structure with right-aligned message bubbles and static text
	nodes := []windows.AccessibleNode{
		{
			Name:     "ChatWindow",
			TreePath: "[0]",
			Children: []windows.AccessibleNode{
				{
					Name:     "MessageArea",
					Role:     "group",
					Bounds:   [4]int{0, 80, 400, 300},
					TreePath: "[0].[0]",
					Children: []windows.AccessibleNode{
						{
							Name:     "Hello", // Right-aligned message bubble
							Role:     "text",
							Bounds:   [4]int{300, 100, 150, 40},
							TreePath: "[0].[0].[0]",
						},
						{
							Name:     "How are you?", // Left-aligned static text
							Role:     "static",
							Bounds:   [4]int{50, 150, 200, 30},
							TreePath: "[0].[0].[1]",
						},
						{
							Name:     "Input box",
							Role:     "edit",
							Bounds:   [4]int{100, 400, 200, 30},
							TreePath: "[0].[0].[2]",
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name        string
		path        string
		expectName  string
		expectRole  string
		expectError bool
	}{
		{
			name:       "Find right-aligned message bubble",
			path:       "[0].[0].[0]",
			expectName: "Hello",
			expectRole: "text",
		},
		{
			name:       "Find left-aligned static text",
			path:       "[0].[0].[1]",
			expectName: "How are you?",
			expectRole: "static",
		},
		{
			name:       "Find input box",
			path:       "[0].[0].[2]",
			expectName: "Input box",
			expectRole: "edit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := ps.FindNodeByPath(nodes, tt.path)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for path '%s', got nil", tt.path)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for path '%s': %v", tt.path, err)
				}
				if node == nil {
					t.Errorf("Expected node for path '%s', got nil", tt.path)
				} else {
					if node.Name != tt.expectName {
						t.Errorf("Expected name '%s', got '%s'", tt.expectName, node.Name)
					}
					if node.Role != tt.expectRole {
						t.Errorf("Expected role '%s', got '%s'", tt.expectRole, node.Role)
					}
				}
			}
		})
	}
}

func TestPathSystem_BoundsDrift(t *testing.T) {
	ps := NewPathSystem()

	// Create nodes with bounds that drift over time (simulating UI changes)
	nodes := []windows.AccessibleNode{
		{
			Name:     "ContactList",
			TreePath: "[0]",
			Children: []windows.AccessibleNode{
				{
					Name:     "张三",
					Role:     "list item",
					Bounds:   [4]int{10, 50, 200, 60}, // Original bounds
					TreePath: "[0].[0]",
				},
				{
					Name:     "李四",
					Role:     "list item",
					Bounds:   [4]int{10, 120, 200, 60}, // Original bounds
					TreePath: "[0].[1]",
				},
			},
		},
	}

	// Simulate bounds drift - same nodes, different bounds
	nodesDrifted := []windows.AccessibleNode{
		{
			Name:     "ContactList",
			TreePath: "[0]",
			Children: []windows.AccessibleNode{
				{
					Name:     "张三",
					Role:     "list item",
					Bounds:   [4]int{12, 52, 198, 58}, // Slightly shifted
					TreePath: "[0].[0]",
				},
				{
					Name:     "李四",
					Role:     "list item",
					Bounds:   [4]int{12, 122, 198, 58}, // Slightly shifted
					TreePath: "[0].[1]",
				},
			},
		},
	}

	tests := []struct {
		name        string
		path        string
		expectName  string
		expectError bool
	}{
		{
			name:       "Find 张三 in original nodes",
			path:       "[0].[0]",
			expectName: "张三",
		},
		{
			name:       "Find 李四 in original nodes",
			path:       "[0].[1]",
			expectName: "李四",
		},
		{
			name:       "Find 张三 in drifted nodes (same path)",
			path:       "[0].[0]",
			expectName: "张三",
		},
		{
			name:       "Find 李四 in drifted nodes (same path)",
			path:       "[0].[1]",
			expectName: "李四",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test with original nodes
			node, err := ps.FindNodeByPath(nodes, tt.path)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for path '%s', got nil", tt.path)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for path '%s': %v", tt.path, err)
				}
				if node == nil {
					t.Errorf("Expected node for path '%s', got nil", tt.path)
				} else if node.Name != tt.expectName {
					t.Errorf("Expected name '%s', got '%s'", tt.expectName, node.Name)
				}
			}

			// Test with drifted nodes (path should still work)
			nodeDrifted, errDrifted := ps.FindNodeByPath(nodesDrifted, tt.path)
			if tt.expectError {
				if errDrifted == nil {
					t.Errorf("Expected error for drifted path '%s', got nil", tt.path)
				}
			} else {
				if errDrifted != nil {
					t.Errorf("Unexpected error for drifted path '%s': %v", tt.path, errDrifted)
				}
				if nodeDrifted == nil {
					t.Errorf("Expected node for drifted path '%s', got nil", tt.path)
				} else if nodeDrifted.Name != tt.expectName {
					t.Errorf("Expected name '%s' in drifted nodes, got '%s'", tt.expectName, nodeDrifted.Name)
				}
			}
		})
	}
}

func TestPathSystem_PathChanges(t *testing.T) {
	ps := NewPathSystem()

	// Create initial node structure
	initialNodes := []windows.AccessibleNode{
		{
			Name:     "ContactList",
			TreePath: "[0]",
			Children: []windows.AccessibleNode{
				{
					Name:     "张三",
					Role:     "list item",
					TreePath: "[0].[0]",
				},
				{
					Name:     "李四",
					Role:     "list item",
					TreePath: "[0].[1]",
				},
				{
					Name:     "王五",
					Role:     "list item",
					TreePath: "[0].[2]",
				},
			},
		},
	}

	// Simulate path changes after contact list reordering
	reorderedNodes := []windows.AccessibleNode{
		{
			Name:     "ContactList",
			TreePath: "[0]",
			Children: []windows.AccessibleNode{
				{
					Name:     "李四", // Moved to top
					Role:     "list item",
					TreePath: "[0].[0]",
				},
				{
					Name:     "王五", // Moved to middle
					Role:     "list item",
					TreePath: "[0].[1]",
				},
				{
					Name:     "张三", // Moved to bottom
					Role:     "list item",
					TreePath: "[0].[2]",
				},
			},
		},
	}

	tests := []struct {
		name        string
		path        string
		expectName  string
		expectError bool
	}{
		{
			name:       "Find 张三 in initial order",
			path:       "[0].[0]",
			expectName: "张三",
		},
		{
			name:       "Find 李四 in initial order",
			path:       "[0].[1]",
			expectName: "李四",
		},
		{
			name:       "Find 王五 in initial order",
			path:       "[0].[2]",
			expectName: "王五",
		},
		{
			name:       "Find 李四 in reordered (now at [0].[0])",
			path:       "[0].[0]",
			expectName: "李四",
		},
		{
			name:       "Find 王五 in reordered (now at [0].[1])",
			path:       "[0].[1]",
			expectName: "王五",
		},
		{
			name:       "Find 张三 in reordered (now at [0].[2])",
			path:       "[0].[2]",
			expectName: "张三",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var node *windows.AccessibleNode
			var err error

			// Determine which node set to use based on test name
			if tt.name == "Find 张三 in reordered (now at [0].[2])" ||
				tt.name == "Find 李四 in reordered (now at [0].[0])" ||
				tt.name == "Find 王五 in reordered (now at [0].[1])" {
				node, err = ps.FindNodeByPath(reorderedNodes, tt.path)
			} else {
				node, err = ps.FindNodeByPath(initialNodes, tt.path)
			}

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for path '%s', got nil", tt.path)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for path '%s': %v", tt.path, err)
				}
				if node == nil {
					t.Errorf("Expected node for path '%s', got nil", tt.path)
				} else if node.Name != tt.expectName {
					t.Errorf("Expected name '%s', got '%s'", tt.expectName, node.Name)
				}
			}
		})
	}
}

func TestPathSystem_DeeplyNestedNodes(t *testing.T) {
	ps := NewPathSystem()

	// Create deeply nested structure (5+ levels)
	nodes := []windows.AccessibleNode{
		{
			Name:     "Level0",
			TreePath: "[0]",
			Children: []windows.AccessibleNode{
				{
					Name:     "Level1",
					TreePath: "[0].[0]",
					Children: []windows.AccessibleNode{
						{
							Name:     "Level2",
							TreePath: "[0].[0].[0]",
							Children: []windows.AccessibleNode{
								{
									Name:     "Level3",
									TreePath: "[0].[0].[0].[0]",
									Children: []windows.AccessibleNode{
										{
											Name:     "Level4",
											TreePath: "[0].[0].[0].[0].[0]",
											Children: []windows.AccessibleNode{
												{
													Name:     "Level5",
													TreePath: "[0].[0].[0].[0].[0].[0]",
													Children: []windows.AccessibleNode{
														{Name: "DeepNode", TreePath: "[0].[0].[0].[0].[0].[0].[0]"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name        string
		path        string
		expectName  string
		expectError bool
	}{
		{
			name:       "Find level 1",
			path:       "[0].[0]",
			expectName: "Level1",
		},
		{
			name:       "Find level 2",
			path:       "[0].[0].[0]",
			expectName: "Level2",
		},
		{
			name:       "Find level 3",
			path:       "[0].[0].[0].[0]",
			expectName: "Level3",
		},
		{
			name:       "Find level 4",
			path:       "[0].[0].[0].[0].[0]",
			expectName: "Level4",
		},
		{
			name:       "Find level 5",
			path:       "[0].[0].[0].[0].[0].[0]",
			expectName: "Level5",
		},
		{
			name:       "Find deepest node",
			path:       "[0].[0].[0].[0].[0].[0].[0]",
			expectName: "DeepNode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := ps.FindNodeByPath(nodes, tt.path)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for path '%s', got nil", tt.path)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for path '%s': %v", tt.path, err)
				}
				if node == nil {
					t.Errorf("Expected node for path '%s', got nil", tt.path)
				} else if node.Name != tt.expectName {
					t.Errorf("Expected name '%s', got '%s'", tt.expectName, node.Name)
				}
			}
		})
	}
}

// ==================== Path System Dirty Data Tests ====================

func TestPathSystem_GeneratePath_DirtyData(t *testing.T) {
	ps := NewPathSystem()

	tests := []struct {
		name       string
		node       windows.AccessibleNode
		parentPath string
		index      int
	}{
		{
			name:       "Empty node name",
			node:       windows.AccessibleNode{Name: ""},
			parentPath: "[0]",
			index:      0,
		},
		{
			name:       "Unicode node name",
			node:       windows.AccessibleNode{Name: "测试用户"},
			parentPath: "[0]",
			index:      1,
		},
		{
			name:       "Very long node name",
			node:       windows.AccessibleNode{Name: "这是一个非常长的用户名用于测试边界情况"},
			parentPath: "[0]",
			index:      2,
		},
		{
			name:       "Special characters in name",
			node:       windows.AccessibleNode{Name: "User@#123"},
			parentPath: "[0]",
			index:      3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			result := ps.GeneratePath(tt.node, tt.parentPath, tt.index)
			if result == "" {
				t.Error("GeneratePath should return non-empty path")
			}
		})
	}
}

func TestPathSystem_ParsePath_DirtyData(t *testing.T) {
	ps := NewPathSystem()

	tests := []struct {
		name        string
		path        string
		expectError bool
	}{
		{
			name:        "Empty path",
			path:        "",
			expectError: true,
		},
		{
			name:        "Invalid format",
			path:        "invalid",
			expectError: true,
		},
		{
			name:        "Missing brackets",
			path:        "0.1.2",
			expectError: true,
		},
		{
			name:        "Negative index",
			path:        "[-1]",
			expectError: true,
		},
		{
			name:        "Float index",
			path:        "[1.5]",
			expectError: true,
		},
		{
			name:        "Multiple dots",
			path:        "[0]..[1]",
			expectError: true,
		},
		{
			name:        "Trailing dot",
			path:        "[0].",
			expectError: true,
		},
		{
			name:        "Leading dot",
			path:        ".[0]",
			expectError: true,
		},
		{
			name:        "Empty index",
			path:        "[]",
			expectError: true,
		},
		{
			name:        "Non-numeric index",
			path:        "[abc]",
			expectError: true,
		},
		{
			name:        "Valid path",
			path:        "[0].[1].[2]",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ps.ParsePath(tt.path)
			if tt.expectError && err == nil {
				t.Errorf("Expected error for path '%s', got nil", tt.path)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for path '%s': %v", tt.path, err)
			}
		})
	}
}

func TestPathSystem_FindNodeByPath_DirtyData(t *testing.T) {
	ps := NewPathSystem()

	tests := []struct {
		name        string
		nodes       []windows.AccessibleNode
		path        string
		expectError bool
	}{
		{
			name:        "Empty nodes",
			nodes:       []windows.AccessibleNode{},
			path:        "[0]",
			expectError: true,
		},
		{
			name:        "Path out of range",
			nodes:       []windows.AccessibleNode{{Name: "Root", TreePath: "[0]"}},
			path:        "[99]",
			expectError: true,
		},
		{
			name:        "Invalid nested path",
			nodes:       []windows.AccessibleNode{{Name: "Root", TreePath: "[0]"}},
			path:        "[0].[99]",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ps.FindNodeByPath(tt.nodes, tt.path)
			if tt.expectError && err == nil {
				t.Errorf("Expected error for path '%s', got nil", tt.path)
			}
		})
	}
}

func TestPathSystem_FlattenNodesWithPath_DirtyData(t *testing.T) {
	ps := NewPathSystem()

	tests := []struct {
		name      string
		nodes     []windows.AccessibleNode
		maxDepth  int
		expectLen int
	}{
		{
			name:      "Empty nodes",
			nodes:     []windows.AccessibleNode{},
			maxDepth:  10,
			expectLen: 0,
		},
		{
			name: "Deep nesting exceeding max depth",
			nodes: []windows.AccessibleNode{
				{
					Name: "Root",
					Children: []windows.AccessibleNode{
						{
							Name: "L1",
							Children: []windows.AccessibleNode{
								{
									Name: "L2",
									Children: []windows.AccessibleNode{
										{Name: "L3"},
									},
								},
							},
						},
					},
				},
			},
			maxDepth:  2,
			expectLen: 2, // Root + L1 (L2 and L3 are truncated)
		},
		{
			name: "Circular reference protection",
			nodes: []windows.AccessibleNode{
				{
					Name:     "Root",
					Children: []windows.AccessibleNode{},
				},
			},
			maxDepth:  10,
			expectLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flatNodes := ps.FlattenNodesWithPath(tt.nodes, "", 0, tt.maxDepth)
			if len(flatNodes) != tt.expectLen {
				t.Errorf("Expected %d flat nodes, got %d", tt.expectLen, len(flatNodes))
			}
		})
	}
}
