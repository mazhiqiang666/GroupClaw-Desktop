package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/yourorg/auto-customer-service/internal/agent/adapter"
	"github.com/yourorg/auto-customer-service/internal/agent/windows"
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

	// Flatten the node tree
	flatNodes := flattenNodes(nodes, 0, *maxDepth)

	// Find the node by path or index
	var targetNode *windows.AccessibleNode
	var nodeIndex int

	if strings.HasPrefix(nodePath, "[") && strings.HasSuffix(nodePath, "]") {
		// Parse as index path like "[3]" or "[1].[2]"
		indexStr := nodePath[1 : len(nodePath)-1]
		if strings.Contains(indexStr, "].[") {
			// Handle nested path like "[1].[2]"
			parts := strings.Split(indexStr, "].[")
			if len(parts) > 0 {
				indexStr = parts[0]
			}
		}
		index, err := strconv.Atoi(indexStr)
		if err != nil || index < 0 || index >= len(flatNodes) {
			log.Fatalf("Invalid node index: %s (valid range: 0-%d)", nodePath, len(flatNodes)-1)
		}
		targetNode = &flatNodes[index]
		nodeIndex = index
	} else {
		// Try to parse as a simple index
		index, err := strconv.Atoi(nodePath)
		if err == nil && index >= 0 && index < len(flatNodes) {
			targetNode = &flatNodes[index]
			nodeIndex = index
		} else {
			// Try to find by name
			for i, node := range flatNodes {
				if node.Name == nodePath {
					targetNode = &node
					nodeIndex = i
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
		fmt.Printf("  Node: [%d] %s (%s)\n", nodeIndex, targetNode.Name, targetNode.Role)
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
