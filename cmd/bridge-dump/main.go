package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/mazhiqiang666/GroupClaw-Desktop/internal/agent/adapter"
	"github.com/mazhiqiang666/GroupClaw-Desktop/internal/agent/windows"
)

var (
	jsonOutput = flag.Bool("json", false, "Output as JSON")
	maxDepth   = flag.Int("depth", 5, "Maximum recursion depth for node traversal")
	filterRole = flag.String("role", "", "Filter nodes by role (e.g., 'list item')")
	filterName = flag.String("name", "", "Filter nodes by name (substring match)")
)

func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
		printUsage()
		return
	}

	command := args[0]
	bridge := windows.NewBridge()

	// 初始化 bridge
	result := bridge.Initialize()
	if result.Status != adapter.StatusSuccess {
		log.Fatalf("Failed to initialize bridge: %s", result.Error)
	}
	defer bridge.Release()

	switch command {
	case "find-wechat":
		findWeChat(bridge)
	case "window-info":
		if len(args) < 2 {
			log.Fatal("Usage: bridge-dump window-info <window-handle>")
		}
		handle, err := strconv.ParseUint(args[1], 10, 64)
		if err != nil {
			log.Fatalf("Invalid window handle: %v", err)
		}
		windowInfo(bridge, uintptr(handle))
	case "list-nodes":
		if len(args) < 2 {
			log.Fatal("Usage: bridge-dump list-nodes <window-handle>")
		}
		handle, err := strconv.ParseUint(args[1], 10, 64)
		if err != nil {
			log.Fatalf("Invalid window handle: %v", err)
		}
		listNodes(bridge, uintptr(handle))
	case "focus":
		if len(args) < 2 {
			log.Fatal("Usage: bridge-dump focus <window-handle>")
		}
		handle, err := strconv.ParseUint(args[1], 10, 64)
		if err != nil {
			log.Fatalf("Invalid window handle: %v", err)
		}
		focusWindow(bridge, uintptr(handle))
	case "click-verify":
		if len(args) < 4 {
			log.Fatal("Usage: bridge-dump click-verify <window-handle> <x> <y>")
		}
		handle, err := strconv.ParseUint(args[1], 10, 64)
		if err != nil {
			log.Fatalf("Invalid window handle: %v", err)
		}
		x, err := strconv.Atoi(args[2])
		if err != nil {
			log.Fatalf("Invalid X coordinate: %v", err)
		}
		y, err := strconv.Atoi(args[3])
		if err != nil {
			log.Fatalf("Invalid Y coordinate: %v", err)
		}
		clickVerify(bridge, uintptr(handle), x, y)
	case "click-node":
		if len(args) < 3 {
			log.Fatal("Usage: bridge-dump click-node <window-handle> <node-path-or-index>")
		}
		handle, err := strconv.ParseUint(args[1], 10, 64)
		if err != nil {
			log.Fatalf("Invalid window handle: %v", err)
		}
		nodePath := args[2]
		clickNode(bridge, uintptr(handle), nodePath)
	case "diagnose":
		if len(args) < 2 {
			log.Fatal("Usage: bridge-dump diagnose <window-handle>")
		}
		handle, err := strconv.ParseUint(args[1], 10, 64)
		if err != nil {
			log.Fatalf("Invalid window handle: %v", err)
		}
		diagnoseBridge(bridge, uintptr(handle))
	case "debug-windows":
		debugWindows(bridge)
	case "debug-accessible":
		if len(args) < 2 {
			log.Fatal("Usage: bridge-dump debug-accessible <window-handle>")
		}
		handle, err := strconv.ParseUint(args[1], 10, 64)
		if err != nil {
			log.Fatalf("Invalid window handle: %v", err)
		}
		debugAccessible(bridge, uintptr(handle))
	case "debug-nodes":
		if len(args) < 2 {
			log.Fatal("Usage: bridge-dump debug-nodes <window-handle> [count]")
		}
		handle, err := strconv.ParseUint(args[1], 10, 64)
		if err != nil {
			log.Fatalf("Invalid window handle: %v", err)
		}
		count := 20
		if len(args) >= 3 {
			count, err = strconv.Atoi(args[2])
			if err != nil {
				count = 20
			}
		}
		debugNodes(bridge, uintptr(handle), count)
	case "debug-uia":
		if len(args) < 2 {
			log.Fatal("Usage: bridge-dump debug-uia <window-handle> [count]")
		}
		handle, err := strconv.ParseUint(args[1], 10, 64)
		if err != nil {
			log.Fatalf("Invalid window handle: %v", err)
		}
		count := 20
		if len(args) >= 3 {
			count, err = strconv.Atoi(args[2])
			if err != nil {
				count = 20
			}
		}
		debugUIA(bridge, uintptr(handle), count)
	default:
		printUsage()
	}
}

func printUsage() {
	fmt.Println("bridge-dump - Windows bridge diagnostic tools")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  bridge-dump find-wechat              - Find WeChat window(s)")
	fmt.Println("  bridge-dump window-info <handle>     - Get window information")
	fmt.Println("  bridge-dump list-nodes <handle>      - List accessibility nodes")
	fmt.Println("  bridge-dump focus <handle>           - Focus window")
	fmt.Println("  bridge-dump click-verify <h> <x> <y> - Click and verify (experimental)")
	fmt.Println("  bridge-dump click-node <h> <path>    - Click node by path/index")
	fmt.Println("  bridge-dump diagnose <handle>        - Comprehensive bridge diagnostics")
	fmt.Println("  bridge-dump debug-windows            - Debug: List all detected windows")
	fmt.Println("  bridge-dump debug-accessible <handle> - Debug: Accessible object diagnostics")
	fmt.Println("  bridge-dump debug-nodes <handle> [N] - Debug: First N nodes with detailed info")
	fmt.Println("  bridge-dump debug-uia <handle> [N]   - Debug: First N UIA nodes")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --json                              - Output as JSON")
	fmt.Println("  --depth <n>                         - Maximum recursion depth (default: 5)")
	fmt.Println("  --role <role>                       - Filter nodes by role")
	fmt.Println("  --name <name>                       - Filter nodes by name (substring)")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  bridge-dump find-wechat")
	fmt.Println("  bridge-dump window-info 123456")
	fmt.Println("  bridge-dump list-nodes 123456 --json --depth 3")
	fmt.Println("  bridge-dump list-nodes 123456 --role \"list item\"")
	fmt.Println("  bridge-dump list-nodes 123456 --name \"张三\"")
	fmt.Println("  bridge-dump focus 123456")
	fmt.Println("  bridge-dump click-verify 123456 100 200")
	fmt.Println("  bridge-dump click-node 123456 \"[3]\"")
	fmt.Println("  bridge-dump click-node 123456 \"[1].[2]\"")
	fmt.Println("  bridge-dump diagnose 123456          - Comprehensive bridge diagnostics")
}

func findWeChat(bridge windows.BridgeInterface) {
	if *jsonOutput {
		findWeChatJSON(bridge)
		return
	}

	fmt.Println("Searching for WeChat windows...")

	// Try to find by title
	handles, result := bridge.FindTopLevelWindows("", "微信")
	if result.Status != adapter.StatusSuccess {
		log.Printf("Failed to find by title: %s", result.Error)
	}

	// Also try by class name
	handles2, result := bridge.FindTopLevelWindows("WeChatMainWndForPC", "")
	if result.Status == adapter.StatusSuccess {
		handles = append(handles, handles2...)
	}

	if len(handles) == 0 {
		fmt.Println("No WeChat windows found")
		return
	}

	fmt.Printf("Found %d WeChat window(s):\n", len(handles))
	for i, handle := range handles {
		info, infoResult := bridge.GetWindowInfo(handle)
		if infoResult.Status == adapter.StatusSuccess {
			fmt.Printf("  [%d] Handle: %d, Class: %s, Title: %s\n",
				i+1, handle, info.Class, info.Title)
		} else {
			fmt.Printf("  [%d] Handle: %d (failed to get info)\n", i+1, handle)
		}
	}
}

func findWeChatJSON(bridge windows.BridgeInterface) {
	// Try to find by title
	handles, result := bridge.FindTopLevelWindows("", "微信")
	if result.Status != adapter.StatusSuccess {
		log.Printf("Failed to find by title: %s", result.Error)
	}

	// Also try by class name
	handles2, result := bridge.FindTopLevelWindows("WeChatMainWndForPC", "")
	if result.Status == adapter.StatusSuccess {
		handles = append(handles, handles2...)
	}

	windowsData := []map[string]interface{}{}
	for _, handle := range handles {
		info, infoResult := bridge.GetWindowInfo(handle)
		if infoResult.Status == adapter.StatusSuccess {
			windowsData = append(windowsData, map[string]interface{}{
				"handle": handle,
				"class":  info.Class,
				"title":  info.Title,
			})
		}
	}

	data := map[string]interface{}{
		"windows": windowsData,
		"count":   len(windowsData),
	}

	jsonData, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(jsonData))
}

func windowInfo(bridge windows.BridgeInterface, handle uintptr) {
	info, result := bridge.GetWindowInfo(handle)
	if result.Status != adapter.StatusSuccess {
		log.Fatalf("Failed to get window info: %s", result.Error)
	}

	data := map[string]interface{}{
		"handle": handle,
		"class":  info.Class,
		"title":  info.Title,
	}

	if *jsonOutput {
		jsonData, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(jsonData))
	} else {
		fmt.Printf("Handle: %d\n", handle)
		fmt.Printf("Class:  %s\n", info.Class)
		fmt.Printf("Title:  %s\n", info.Title)
	}
}

func listNodes(bridge windows.BridgeInterface, handle uintptr) {
	nodes, result := bridge.EnumerateAccessibleNodes(handle)
	if result.Status != adapter.StatusSuccess {
		log.Fatalf("Failed to enumerate nodes: %s", result.Error)
	}

	if *jsonOutput {
		printNodesJSON(nodes, 0)
	} else {
		fmt.Printf("Found %d accessibility node(s) (max depth: %d):\n", len(nodes), *maxDepth)
		if *filterRole != "" || *filterName != "" {
			fmt.Printf("Filtering by role='%s' name='%s'\n", *filterRole, *filterName)
		}
		printNodesText(nodes, 0, "")
	}
}

func printNodesText(nodes []windows.AccessibleNode, depth int, path string) {
	if depth >= *maxDepth {
		return
	}

	indent := ""
	for i := 0; i < depth; i++ {
		indent += "  "
	}

	for i, node := range nodes {
		// Apply filters
		if *filterRole != "" && node.Role != *filterRole {
			continue
		}
		if *filterName != "" && !contains(node.Name, *filterName) {
			continue
		}

		// Build node path
		nodePath := path
		if nodePath == "" {
			nodePath = fmt.Sprintf("[%d]", i+1)
		} else {
			nodePath = fmt.Sprintf("%s.[%d]", nodePath, i+1)
		}

		// Print node with bounds if available
		boundsStr := ""
		if len(node.Bounds) == 4 {
			boundsStr = fmt.Sprintf(" Bounds(x=%d,y=%d,w=%d,h=%d)",
				node.Bounds[0], node.Bounds[1], node.Bounds[2], node.Bounds[3])
		}

		fmt.Printf("%s%s Handle: %d, Name: %s, Role: %s, Class: %s%s\n",
			indent, nodePath, node.Handle, node.Name, node.Role, node.ClassName, boundsStr)

		if len(node.Children) > 0 {
			printNodesText(node.Children, depth+1, nodePath)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && len(s) >= len(substr) &&
		(s == substr || (len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findSubstring(s, substr))))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func printNodesJSON(nodes []windows.AccessibleNode, depth int) interface{} {
	if depth >= *maxDepth {
		return nil
	}

	result := []map[string]interface{}{}
	for _, node := range nodes {
		nodeData := map[string]interface{}{
			"handle":    node.Handle,
			"name":      node.Name,
			"role":      node.Role,
			"className": node.ClassName,
		}

		if len(node.Bounds) == 4 {
			nodeData["bounds"] = map[string]interface{}{
				"x":      node.Bounds[0],
				"y":      node.Bounds[1],
				"width":  node.Bounds[2],
				"height": node.Bounds[3],
			}
		}

		if len(node.Children) > 0 {
			children := printNodesJSON(node.Children, depth+1)
			if children != nil {
				nodeData["children"] = children
			}
		}

		result = append(result, nodeData)
	}

	if depth == 0 {
		// Top level: wrap in object
		data := map[string]interface{}{
			"nodes": result,
			"count": len(result),
		}
		jsonData, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(jsonData))
		return nil
	}

	return result
}

func focusWindow(bridge windows.BridgeInterface, handle uintptr) {
	result := bridge.FocusWindow(handle)
	if result.Status != adapter.StatusSuccess {
		log.Fatalf("Failed to focus window: %s", result.Error)
	}

	if *jsonOutput {
		data := map[string]interface{}{
			"success": true,
			"handle":  handle,
		}
		jsonData, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(jsonData))
	} else {
		fmt.Printf("Successfully focused window: %d\n", handle)
	}
}

func clickVerify(bridge windows.BridgeInterface, handle uintptr, x, y int) {
	// Focus the window first
	focusResult := bridge.FocusWindow(handle)
	if focusResult.Status != adapter.StatusSuccess {
		log.Fatalf("Failed to focus window: %s", focusResult.Error)
	}

	// Click at the specified coordinates
	clickResult := bridge.Click(handle, x, y)
	if clickResult.Status != adapter.StatusSuccess {
		log.Fatalf("Failed to click at (%d, %d): %s", x, y, clickResult.Error)
	}

	// Capture screenshot for verification
	screenshot, captureResult := bridge.CaptureWindow(handle)
	if captureResult.Status != adapter.StatusSuccess {
		log.Printf("Warning: Failed to capture screenshot: %s", captureResult.Error)
	}

	if *jsonOutput {
		data := map[string]interface{}{
			"success":      true,
			"handle":       handle,
			"click_x":      x,
			"click_y":      y,
			"screenshot":   len(screenshot) > 0,
			"screenshot_size": len(screenshot),
		}
		jsonData, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(jsonData))
	} else {
		fmt.Printf("Click verification completed:\n")
		fmt.Printf("  Window: %d\n", handle)
		fmt.Printf("  Position: (%d, %d)\n", x, y)
		fmt.Printf("  Screenshot captured: %v (size: %d bytes)\n", len(screenshot) > 0, len(screenshot))
	}
}

func clickNode(bridge windows.BridgeInterface, handle uintptr, nodePath string) {
	// Focus the window first
	focusResult := bridge.FocusWindow(handle)
	if focusResult.Status != adapter.StatusSuccess {
		log.Fatalf("Failed to focus window: %s", focusResult.Error)
	}

	// Enumerate accessible nodes
	nodes, result := bridge.EnumerateAccessibleNodes(handle)
	if result.Status != adapter.StatusSuccess {
		log.Fatalf("Failed to enumerate nodes: %s", result.Error)
	}

	// Build path map for hierarchical lookup
	pathMap := make(map[string]*windows.AccessibleNode)
	flatNodes := flattenNodesWithPath(nodes, nil, 0, *maxDepth, pathMap, "")

	// Find the node by hierarchical path or index
	var targetNode *windows.AccessibleNode
	var nodeIndex int
	var foundPath string

	// Check if path is in the format [0].[3].[2] (hierarchical)
	if strings.Contains(nodePath, "].[") {
		// Hierarchical path lookup
		if node, ok := pathMap[nodePath]; ok {
			targetNode = node
			foundPath = nodePath
		}
	} else if strings.HasPrefix(nodePath, "[") && strings.HasSuffix(nodePath, "]") {
		// Simple index path like "[3]"
		indexStr := nodePath[1 : len(nodePath)-1]
		index, err := strconv.Atoi(indexStr)
		if err == nil && index >= 0 && index < len(flatNodes) {
			targetNode = &flatNodes[index]
			nodeIndex = index
			foundPath = fmt.Sprintf("[%d]", index)
		}
	} else {
		// Try to parse as a simple index
		index, err := strconv.Atoi(nodePath)
		if err == nil && index >= 0 && index < len(flatNodes) {
			targetNode = &flatNodes[index]
			nodeIndex = index
			foundPath = fmt.Sprintf("[%d]", index)
		} else {
			// Try to find by name
			for i, node := range flatNodes {
				if node.Name == nodePath {
					targetNode = &node
					nodeIndex = i
					foundPath = fmt.Sprintf("[%d]", i)
					break
				}
			}
		}
	}

	if targetNode == nil {
		log.Fatalf("Node not found: %s (searched %d nodes)", nodePath, len(flatNodes))
	}

	// Calculate center of node bounds
	if len(targetNode.Bounds) != 4 {
		log.Fatalf("Node has invalid bounds: %v", targetNode.Bounds)
	}

	bounds := targetNode.Bounds
	clickX := bounds[0] + bounds[2]/2
	clickY := bounds[1] + bounds[3]/2

	// Click at the node center
	clickResult := bridge.Click(handle, clickX, clickY)
	if clickResult.Status != adapter.StatusSuccess {
		log.Fatalf("Failed to click at (%d, %d): %s", clickX, clickY, clickResult.Error)
	}

	// Capture screenshot for verification
	screenshot, captureResult := bridge.CaptureWindow(handle)
	if captureResult.Status != adapter.StatusSuccess {
		log.Printf("Warning: Failed to capture screenshot: %s", captureResult.Error)
	}

	if *jsonOutput {
		data := map[string]interface{}{
			"success":         true,
			"handle":          handle,
			"node_index":      nodeIndex,
			"node_path":       foundPath,
			"node_name":       targetNode.Name,
			"node_role":       targetNode.Role,
			"node_bounds":     bounds,
			"click_x":         clickX,
			"click_y":         clickY,
			"screenshot":      len(screenshot) > 0,
			"screenshot_size": len(screenshot),
		}
		jsonData, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(jsonData))
	} else {
		fmt.Printf("Node click completed:\n")
		fmt.Printf("  Window: %d\n", handle)
		fmt.Printf("  Node: %s [%d] %s (%s)\n", foundPath, nodeIndex, targetNode.Name, targetNode.Role)
		fmt.Printf("  Bounds: x=%d, y=%d, w=%d, h=%d\n", bounds[0], bounds[1], bounds[2], bounds[3])
		fmt.Printf("  Click position: (%d, %d)\n", clickX, clickY)
		fmt.Printf("  Screenshot captured: %v (size: %d bytes)\n", len(screenshot) > 0, len(screenshot))
	}
}

// flattenNodes recursively flattens AccessibleNode tree
func flattenNodes(nodes []windows.AccessibleNode, depth int, maxDepth int) []windows.AccessibleNode {
	if depth >= maxDepth {
		return nodes
	}

	result := make([]windows.AccessibleNode, 0, len(nodes))
	for _, node := range nodes {
		result = append(result, node)
		if len(node.Children) > 0 {
			result = append(result, flattenNodes(node.Children, depth+1, maxDepth)...)
		}
	}
	return result
}

// flattenNodesWithPath recursively flattens AccessibleNode tree and builds path map
func flattenNodesWithPath(nodes []windows.AccessibleNode, parent *windows.AccessibleNode, depth int, maxDepth int, pathMap map[string]*windows.AccessibleNode, parentPath string) []windows.AccessibleNode {
	if depth >= maxDepth {
		return nodes
	}

	result := make([]windows.AccessibleNode, 0, len(nodes))
	for i, node := range nodes {
		// Build hierarchical path
		nodePath := parentPath
		if nodePath == "" {
			nodePath = fmt.Sprintf("[%d]", i)
		} else {
			nodePath = fmt.Sprintf("%s.[%d]", nodePath, i)
		}

		// Store in path map for hierarchical lookup
		pathMap[nodePath] = &node

		result = append(result, node)
		if len(node.Children) > 0 {
			result = append(result, flattenNodesWithPath(node.Children, &node, depth+1, maxDepth, pathMap, nodePath)...)
		}
	}
	return result
}

// diagnoseBridge 执行综合桥接诊断
func diagnoseBridge(bridge windows.BridgeInterface, handle uintptr) {
	fmt.Printf("=== Bridge Diagnostics for Handle: %d ===\n\n", handle)

	// 1. 获取窗口信息
	fmt.Println("1. Window Information:")
	info, infoResult := bridge.GetWindowInfo(handle)
	if infoResult.Status == adapter.StatusSuccess {
		fmt.Printf("   Class: %s\n", info.Class)
		fmt.Printf("   Title: %s\n", info.Title)
		fmt.Printf("   Status: %s\n", infoResult.Status)
	} else {
		fmt.Printf("   ERROR: %s\n", infoResult.Error)
	}
	fmt.Println()

	// 2. 获取可访问对象诊断
	fmt.Println("2. GetAccessible Diagnostics:")
	_, accResult := bridge.GetAccessible(handle)
	if accResult.Status == adapter.StatusSuccess {
		fmt.Printf("   Accessible object obtained: YES\n")
		if len(accResult.Diagnostics) > 0 {
			fmt.Printf("   Diagnostics from GetAccessible:\n")
			for _, diag := range accResult.Diagnostics {
				fmt.Printf("     - %s: ", diag.Message)
				for k, v := range diag.Context {
					fmt.Printf("%s=%s ", k, v)
				}
				fmt.Println()
			}
		}
	} else {
		fmt.Printf("   Accessible object obtained: NO\n")
		fmt.Printf("   Error: %s\n", accResult.Error)
		fmt.Printf("   Reason Code: %s\n", accResult.ReasonCode)
		if len(accResult.Diagnostics) > 0 {
			fmt.Printf("   Diagnostics:\n")
			for _, diag := range accResult.Diagnostics {
				fmt.Printf("     - %s: ", diag.Message)
				for k, v := range diag.Context {
					fmt.Printf("%s=%s ", k, v)
				}
				fmt.Println()
			}
		}
	}
	fmt.Println()

	// 3. 枚举可访问节点
	fmt.Println("3. EnumerateAccessibleNodes Diagnostics:")
	nodes, enumResult := bridge.EnumerateAccessibleNodes(handle)
	if enumResult.Status == adapter.StatusSuccess {
		fmt.Printf("   Total nodes returned: %d\n", len(nodes))

		// 显示第一个节点（通常是根节点）的信息
		if len(nodes) > 0 {
			root := nodes[0]
			fmt.Printf("   Root node:\n")
			fmt.Printf("     Name: %s\n", root.Name)
			fmt.Printf("     Role: %s\n", root.Role)
			fmt.Printf("     Class: %s\n", root.ClassName)
			fmt.Printf("     Children count: %d\n", len(root.Children))
			if len(root.Bounds) == 4 {
				fmt.Printf("     Bounds: x=%d, y=%d, w=%d, h=%d\n",
					root.Bounds[0], root.Bounds[1], root.Bounds[2], root.Bounds[3])
			}
		}

		// 显示枚举诊断信息
		if len(enumResult.Diagnostics) > 0 {
			fmt.Printf("   Enumeration Diagnostics:\n")
			for _, diag := range enumResult.Diagnostics {
				fmt.Printf("     - %s: ", diag.Message)
				for k, v := range diag.Context {
					fmt.Printf("%s=%s ", k, v)
				}
				fmt.Println()
			}
		}

		// 扁平化所有节点进行简单分析
		flatNodes := flattenNodes(nodes, 0, *maxDepth)
		fmt.Printf("   Flattened nodes count: %d\n", len(flatNodes))

		// 统计角色分布
		roleCount := make(map[string]int)
		for _, node := range flatNodes {
			roleCount[node.Role]++
		}

		if len(roleCount) > 0 {
			fmt.Printf("   Role distribution:\n")
			for role, count := range roleCount {
				fmt.Printf("     %s: %d\n", role, count)
			}
		}

		// 检查是否有列表项节点
		listItemCount := 0
		for _, node := range flatNodes {
			if node.Role == "listitem" || node.Role == "list item" || strings.Contains(strings.ToLower(node.Role), "list") {
				listItemCount++
				if listItemCount <= 3 {
					fmt.Printf("   List item found: name='%s', role='%s'\n", node.Name, node.Role)
				}
			}
		}
		if listItemCount > 0 {
			fmt.Printf("   Total list items found: %d\n", listItemCount)
		}

	} else {
		fmt.Printf("   Enumeration failed: %s\n", enumResult.Error)
		if len(enumResult.Diagnostics) > 0 {
			fmt.Printf("   Diagnostics:\n")
			for _, diag := range enumResult.Diagnostics {
				fmt.Printf("     - %s: ", diag.Message)
				for k, v := range diag.Context {
					fmt.Printf("%s=%s ", k, v)
				}
				fmt.Println()
			}
		}
	}
	fmt.Println()

	// 4. 诊断总结
	fmt.Println("4. Diagnostic Summary:")
	fmt.Printf("   Window handle: %d\n", handle)
	fmt.Printf("   Window class: %s\n", info.Class)
	fmt.Printf("   Window title: %s\n", info.Title)
	fmt.Printf("   GetAccessible succeeded: %v\n", accResult.Status == adapter.StatusSuccess)

	// 确定问题类型
	if accResult.Status != adapter.StatusSuccess {
		fmt.Printf("   PROBLEM: Cannot get accessible subtree\n")
		fmt.Printf("   Reason: %s (code: %s)\n", accResult.Error, accResult.ReasonCode)
		fmt.Printf("   Recommendation: Try different OBJID or child window\n")
	} else if len(nodes) == 0 || (len(nodes) == 1 && len(nodes[0].Children) == 0) {
		fmt.Printf("   PROBLEM: Got accessible subtree but no useful nodes\n")
		fmt.Printf("   Recommendation: Check if app implements MSAA properly\n")
	} else {
		flatNodes := flattenNodes(nodes, 0, *maxDepth)
		fmt.Printf("   STATUS: Got accessible subtree with %d nodes\n", len(flatNodes))
		fmt.Printf("   Recommendation: Check rule filtering logic\n")
	}

	fmt.Println("\n=== End of Diagnostics ===")
}

// debugWindows 调试窗口列表
func debugWindows(bridge windows.BridgeInterface) {
	fmt.Println("=== Debug: Listing All Windows ===")

	// 查找所有顶级窗口
	handles, result := bridge.FindTopLevelWindows("", "")
	if result.Status != adapter.StatusSuccess {
		log.Printf("Failed to enumerate windows: %s", result.Error)
		return
	}

	fmt.Printf("Found %d top-level window(s):\n", len(handles))
	wechatWindows := []uintptr{}
	for i, handle := range handles {
		info, infoResult := bridge.GetWindowInfo(handle)
		if infoResult.Status == adapter.StatusSuccess {
			// 检查是否是微信窗口
			isWeChat := info.Class == "WeChatMainWndForPC" || info.Title == "微信"
			wechatMarker := ""
			if isWeChat {
				wechatMarker = " [WECHAT]"
				wechatWindows = append(wechatWindows, handle)
			}

			fmt.Printf("  [%d] Handle: 0x%X (%d), Class: %s, Title: %s%s\n",
				i+1, handle, handle, info.Class, info.Title, wechatMarker)
		} else {
			fmt.Printf("  [%d] Handle: 0x%X (%d) [failed to get info]\n", i+1, handle, handle)
		}
	}

	// 显示微信窗口的详细信息
	if len(wechatWindows) > 0 {
		fmt.Printf("\n=== WeChat Windows Details ===\n")
		for i, handle := range wechatWindows {
			info, infoResult := bridge.GetWindowInfo(handle)
			if infoResult.Status == adapter.StatusSuccess {
				fmt.Printf("WeChat Window [%d]:\n", i+1)
				fmt.Printf("  Handle: 0x%X (%d)\n", handle, handle)
				fmt.Printf("  Class: %s\n", info.Class)
				fmt.Printf("  Title: %s\n", info.Title)

				// 尝试获取可访问对象
				fmt.Printf("  Testing GetAccessible...\n")
				_, accResult := bridge.GetAccessible(handle)
				if accResult.Status == adapter.StatusSuccess {
					fmt.Printf("    SUCCESS: Accessible object obtained\n")
				} else {
					fmt.Printf("    FAILED: %s\n", accResult.Error)
				}
				fmt.Println()
			}
		}
	}

	fmt.Println("\n=== Debug Windows Complete ===")
}

// debugAccessible 调试可访问对象
func debugAccessible(bridge windows.BridgeInterface, handle uintptr) {
	fmt.Printf("=== Debug: Accessible Diagnostics for Handle: 0x%X (%d) ===\n\n", handle, handle)

	// 获取窗口信息
	info, infoResult := bridge.GetWindowInfo(handle)
	if infoResult.Status != adapter.StatusSuccess {
		fmt.Printf("ERROR: Failed to get window info: %s\n", infoResult.Error)
		return
	}

	fmt.Printf("Window Information:\n")
	fmt.Printf("  Handle: 0x%X (%d)\n", handle, handle)
	fmt.Printf("  Class: %s\n", info.Class)
	fmt.Printf("  Title: %s\n", info.Title)
	fmt.Println()

	// 尝试获取可访问对象
	fmt.Printf("Attempting GetAccessible...\n")
	_, result := bridge.GetAccessible(handle)
	if result.Status != adapter.StatusSuccess {
		fmt.Printf("  FAILED: %s (code: %s)\n", result.Error, result.ReasonCode)

		// 显示详细诊断信息
		if len(result.Diagnostics) > 0 {
			fmt.Printf("  Diagnostics:\n")
			for _, diag := range result.Diagnostics {
				fmt.Printf("    - %s\n", diag.Message)
				for k, v := range diag.Context {
					fmt.Printf("      %s: %s\n", k, v)
				}
			}
		}

		// 建议尝试子窗口
		fmt.Printf("\nSuggestion: Try child windows if parent window fails.\n")
		fmt.Printf("Use 'bridge-dump debug-windows' to see child window handles.\n")
		fmt.Printf("Then use 'bridge-dump debug-accessible <child-handle>' to test.\n")
	} else {
		fmt.Printf("  SUCCESS: Accessible object obtained\n")

		// 显示诊断信息
		if len(result.Diagnostics) > 0 {
			fmt.Printf("  Diagnostics:\n")
			for _, diag := range result.Diagnostics {
				fmt.Printf("    - %s\n", diag.Message)
				for k, v := range diag.Context {
					fmt.Printf("      %s: %s\n", k, v)
				}
			}
		}

		// 通过 EnumerateAccessibleNodes 获取更多信息
		fmt.Printf("\nEnumerating accessible nodes for more details...\n")
		nodes, enumResult := bridge.EnumerateAccessibleNodes(handle)
		if enumResult.Status == adapter.StatusSuccess {
			fmt.Printf("  Enumeration succeeded: found %d top-level node(s)\n", len(nodes))
			if len(nodes) > 0 {
				root := nodes[0]
				fmt.Printf("  Root node children: %d\n", len(root.Children))

				// 扁平化显示前几个节点
				flatNodes := flattenNodes(nodes, 0, 3)
				displayCount := 5
				if len(flatNodes) < displayCount {
					displayCount = len(flatNodes)
				}

				if displayCount > 0 {
					fmt.Printf("  First %d nodes:\n", displayCount)
					for i := 0; i < displayCount; i++ {
						node := flatNodes[i]
						fmt.Printf("    [%d] Name: %s, Role: %s\n", i, node.Name, node.Role)
					}
				}
			}
		} else {
			fmt.Printf("  Enumeration failed: %s\n", enumResult.Error)
		}
	}

	fmt.Println("\n=== Debug Accessible Complete ===")
}

// debugNodes 调试节点信息
func debugNodes(bridge windows.BridgeInterface, handle uintptr, count int) {
	fmt.Printf("=== Debug: First %d Nodes for Handle: 0x%X (%d) ===\n\n", count, handle, handle)

	// 枚举节点
	nodes, result := bridge.EnumerateAccessibleNodes(handle)
	if result.Status != adapter.StatusSuccess {
		fmt.Printf("ERROR: Failed to enumerate nodes: %s\n", result.Error)

		// 显示诊断信息
		if len(result.Diagnostics) > 0 {
			fmt.Printf("Diagnostics:\n")
			for _, diag := range result.Diagnostics {
				fmt.Printf("  - %s\n", diag.Message)
				for k, v := range diag.Context {
					fmt.Printf("    %s: %s\n", k, v)
				}
			}
		}
		return
	}

	// 显示诊断信息
	if len(result.Diagnostics) > 0 {
		fmt.Printf("Enumeration Diagnostics:\n")
		for _, diag := range result.Diagnostics {
			fmt.Printf("  - %s\n", diag.Message)
			for k, v := range diag.Context {
				fmt.Printf("    %s: %s\n", k, v)
			}
		}
		fmt.Println()
	}

	// 扁平化所有节点
	flatNodes := flattenNodes(nodes, 0, 10) // 使用较大的深度

	fmt.Printf("Total nodes found: %d\n", len(flatNodes))
	fmt.Printf("Showing first %d nodes:\n\n", count)

	displayCount := count
	if len(flatNodes) < displayCount {
		displayCount = len(flatNodes)
	}

	for i := 0; i < displayCount; i++ {
		node := flatNodes[i]
		fmt.Printf("Node [%d]:\n", i)
		fmt.Printf("  Handle: %d\n", node.Handle)
		fmt.Printf("  Name: %s\n", node.Name)
		fmt.Printf("  Role: %s\n", node.Role)
		fmt.Printf("  Class: %s\n", node.ClassName)
		if len(node.Bounds) == 4 {
			fmt.Printf("  Bounds: x=%d, y=%d, w=%d, h=%d\n",
				node.Bounds[0], node.Bounds[1], node.Bounds[2], node.Bounds[3])
		}

		// 构建树路径（简化）
		fmt.Printf("  Tree path: ")
		if i == 0 {
			fmt.Printf("root")
		} else {
			// 简单索引路径
			fmt.Printf("[%d]", i)
		}

		fmt.Printf("\n\n")
	}

	if len(flatNodes) == 0 {
		fmt.Printf("WARNING: No nodes found. This indicates bridge layer issue.\n")
		fmt.Printf("Diagnostic summary:\n")
		fmt.Printf("  - Accessible object obtained: %v\n",
			containsDiagnostic(result.Diagnostics, "accessible_obtained", "true"))
		fmt.Printf("  - Child count: %s\n",
			getDiagnosticValue(result.Diagnostics, "child_count", "0"))
		fmt.Printf("  - Bridge layer blocked: %s\n",
			getDiagnosticValue(result.Diagnostics, "bridge_layer_blocked", "unknown"))
	}

	fmt.Println("=== Debug Nodes Complete ===")
}

// containsDiagnostic 检查诊断中是否包含特定键值
func containsDiagnostic(diagnostics []adapter.Diagnostic, key, value string) bool {
	for _, diag := range diagnostics {
		if diag.Context != nil {
			if val, ok := diag.Context[key]; ok && val == value {
				return true
			}
		}
	}
	return false
}

// getDiagnosticValue 获取诊断中的值
func getDiagnosticValue(diagnostics []adapter.Diagnostic, key, defaultValue string) string {
	for _, diag := range diagnostics {
		if diag.Context != nil {
			if val, ok := diag.Context[key]; ok {
				return val
			}
		}
	}
	return defaultValue
}

// debugUIA 调试UIA节点
func debugUIA(bridge windows.BridgeInterface, handle uintptr, count int) {
	fmt.Printf("=== Debug: UIA First %d Nodes for Handle: 0x%X (%d) ===\n\n", count, handle, handle)

	// 获取窗口信息
	info, infoResult := bridge.GetWindowInfo(handle)
	if infoResult.Status != adapter.StatusSuccess {
		fmt.Printf("ERROR: Failed to get window info: %s\n", infoResult.Error)
		return
	}

	fmt.Printf("Window Information:\n")
	fmt.Printf("  Handle: 0x%X (%d)\n", handle, handle)
	fmt.Printf("  Class: %s\n", info.Class)
	fmt.Printf("  Title: %s\n", info.Title)
	fmt.Println()

	// 创建UIA桥接器
	uiaBridge := windows.NewUIABridge()
	defer uiaBridge.ReleaseUIA()

	// 枚举UIA节点
	nodes, result := uiaBridge.EnumerateUIANodes(handle, count)
	if result.Status != adapter.StatusSuccess {
		fmt.Printf("ERROR: Failed to enumerate UIA nodes: %s\n", result.Error)

		// 显示诊断信息
		if len(result.Diagnostics) > 0 {
			fmt.Printf("Diagnostics:\n")
			for _, diag := range result.Diagnostics {
				fmt.Printf("  - %s\n", diag.Message)
				for k, v := range diag.Context {
					fmt.Printf("    %s: %s\n", k, v)
				}
			}
		}
		return
	}

	// 显示诊断信息
	if len(result.Diagnostics) > 0 {
		fmt.Printf("UIA Enumeration Diagnostics:\n")
		for _, diag := range result.Diagnostics {
			fmt.Printf("  - %s\n", diag.Message)
			for k, v := range diag.Context {
				fmt.Printf("    %s: %s\n", k, v)
			}
		}
		fmt.Println()
	}

	fmt.Printf("Total UIA nodes found: %d\n", len(nodes))
	fmt.Printf("Showing first %d UIA nodes:\n\n", count)

	displayCount := count
	if len(nodes) < displayCount {
		displayCount = len(nodes)
	}

	for i := 0; i < displayCount; i++ {
		node := nodes[i]
		fmt.Printf("UIA Node [%d]:\n", i)
		fmt.Printf("  Name: %s\n", node.Name)
		fmt.Printf("  ControlType: %s\n", node.ControlType)
		fmt.Printf("  AutomationId: %s\n", node.AutomationId)
		fmt.Printf("  ClassName: %s\n", node.ClassName)
		fmt.Printf("  Depth: %d\n", node.Depth)
		if node.Bounds[2] > 0 && node.Bounds[3] > 0 {
			fmt.Printf("  Bounds: left=%d, top=%d, width=%d, height=%d\n",
				node.Bounds[0], node.Bounds[1], node.Bounds[2], node.Bounds[3])
		}
		fmt.Println()
	}

	if len(nodes) == 0 {
		fmt.Printf("WARNING: No UIA nodes found.\n")
		fmt.Printf("Possible reasons:\n")
		fmt.Printf("  1. UIA not supported by this application\n")
		fmt.Printf("  2. UIA initialization failed\n")
		fmt.Printf("  3. Window handle may not represent a valid UI element\n")
		fmt.Printf("\nCheck diagnostics above for more details.\n")
	}

	fmt.Println("=== Debug UIA Complete ===")
}

