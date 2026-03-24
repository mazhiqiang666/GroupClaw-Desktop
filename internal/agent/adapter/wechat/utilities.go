package wechat

import (
	"fmt"
	"strings"

	"github.com/mazhiqiang666/GroupClaw-Desktop/internal/agent/windows"
)

// ==================== Utility Functions ====================

// generateStableKey 生成稳定的定位键（包含多级上下文）
func generateStableKey(node windows.AccessibleNode, parentContext string, treePath string) string {
	if len(node.Bounds) != 4 {
		return ""
	}
	// 格式: tree_path|parent|role|name|x_y_w_h
	key := fmt.Sprintf("%s|%s|%s|%s|%d_%d_%d_%d",
		treePath, parentContext, node.Role, node.Name,
		node.Bounds[0], node.Bounds[1], node.Bounds[2], node.Bounds[3])
	return key
}

// generateParentContext 生成父节点上下文
func generateParentContext(node windows.AccessibleNode) string {
	// 基于节点角色和类名生成父上下文标识
	// 例如: "ContactList|List" 表示在联系人列表中的列表项
	context := ""
	if node.Role == "list item" || node.Role == "ListItem" {
		context = "ListItem"
	} else if strings.Contains(strings.ToLower(node.ClassName), "list") {
		context = "ListContainer"
	}
	return context
}
