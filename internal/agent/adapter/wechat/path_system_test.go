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
