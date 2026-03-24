package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/mazhiqiang666/GroupClaw-Desktop/internal/agent/adapter"
	"github.com/mazhiqiang666/GroupClaw-Desktop/internal/agent/windows"
)

var (
	jsonOutput    = flag.Bool("json", false, "Output as JSON")
	maxDepth      = flag.Int("depth", 5, "Maximum recursion depth for node traversal")
	filterRole    = flag.String("role", "", "Filter nodes by role (e.g., 'list item')")
	filterName    = flag.String("name", "", "Filter nodes by name (substring match)")
	splitRegions  = flag.Bool("split-regions", false, "Split window into regions for OCR (left_sidebar, message_area, input_area)")
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
	case "debug-ocr":
		if len(args) < 2 {
			log.Fatal("Usage: bridge-dump debug-ocr <window-handle> [lang]")
		}
		handle, err := strconv.ParseUint(args[1], 10, 64)
		if err != nil {
			log.Fatalf("Invalid window handle: %v", err)
		}
		lang := "chi_sim" // 默认简体中文
		if len(args) >= 3 {
			lang = args[2]
		}
		debugOCR(bridge, uintptr(handle), lang)
	case "click-conversation":
		if len(args) < 3 {
			log.Fatal("Usage: bridge-dump click-conversation <window-handle> <index> [strategy]")
			log.Fatal("Strategies: rect_center, left_quarter_center, avatar_center, text_center")
		}
		handle, err := strconv.ParseUint(args[1], 10, 64)
		if err != nil {
			log.Fatalf("Invalid window handle: %v", err)
		}
		index, err := strconv.Atoi(args[2])
		if err != nil {
			log.Fatalf("Invalid index: %v", err)
		}
		strategy := ""
		if len(args) >= 4 {
			strategy = args[3]
		}
		clickConversation(bridge, uintptr(handle), index, strategy)
	case "debug-vision":
		if len(args) < 2 {
			log.Fatal("Usage: bridge-dump debug-vision <window-handle>")
		}
		handle, err := strconv.ParseUint(args[1], 10, 64)
		if err != nil {
			log.Fatalf("Invalid window handle: %v", err)
		}
		debugVision(bridge, uintptr(handle))
	case "focus-vision":
		if len(args) < 2 {
			log.Fatal("Usage: bridge-dump focus-vision <window-handle> [index] [click-strategy] [wait-ms]")
			log.Fatal("  index: -1 for default selection (first high-confidence item)")
			log.Fatal("  click-strategy: rect_center, left_quarter_center, avatar_center, text_center, or empty for default")
			log.Fatal("  wait-ms: milliseconds to wait after click (default: 800)")
		}
		handle, err := strconv.ParseUint(args[1], 10, 64)
		if err != nil {
			log.Fatalf("Invalid window handle: %v", err)
		}
		index := -1 // 默认使用自动选择
		if len(args) >= 3 && args[2] != "" {
			indexVal, err := strconv.Atoi(args[2])
			if err == nil {
				index = indexVal
			}
		}
		strategy := ""
		if len(args) >= 4 && args[3] != "" {
			strategy = args[3]
		}
		waitMs := 800 // 默认800ms
		if len(args) >= 5 && args[4] != "" {
			waitVal, err := strconv.Atoi(args[4])
			if err == nil && waitVal > 0 {
				waitMs = waitVal
			}
		}
		focusVision(bridge, uintptr(handle), index, strategy, waitMs)
	case "debug-input-box":
		if len(args) < 2 {
			log.Fatal("Usage: bridge-dump debug-input-box <window-handle>")
		}
		handle, err := strconv.ParseUint(args[1], 10, 64)
		if err != nil {
			log.Fatalf("Invalid window handle: %v", err)
		}
		debugInputBox(bridge, uintptr(handle))
	case "click-input-box":
		if len(args) < 3 {
			log.Fatal("Usage: bridge-dump click-input-box <window-handle> <strategy>")
			log.Fatal("Strategies: input_left_third, input_center, input_left_quarter, input_double_click_center")
		}
		handle, err := strconv.ParseUint(args[1], 10, 64)
		if err != nil {
			log.Fatalf("Invalid window handle: %v", err)
		}
		strategy := args[2]
		clickInputBox(bridge, uintptr(handle), strategy)
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
	fmt.Println("  bridge-dump debug-ocr <handle> [lang] - Debug: OCR text extraction")
	fmt.Println("                                         Use --split-regions for region-based OCR")
	fmt.Println("  bridge-dump debug-vision <handle>     - Debug: Visual conversation detection")
	fmt.Println("  bridge-dump click-conversation <h> <i> - Click conversation by vision detection index")
	fmt.Println("  bridge-dump debug-input-box <handle>   - Debug: Input box detection")
	fmt.Println("  bridge-dump click-input-box <h> <strategy> - Click input box with specified strategy")
	fmt.Println("                                         Strategies: input_left_third, input_center, input_left_quarter, input_double_click_center")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --json                              - Output as JSON")
	fmt.Println("  --depth <n>                         - Maximum recursion depth (default: 5)")
	fmt.Println("  --role <role>                       - Filter nodes by role")
	fmt.Println("  --name <name>                       - Filter nodes by name (substring)")
	fmt.Println("  --split-regions                     - Split window into regions for OCR")
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
	fmt.Println("  bridge-dump click-conversation 123456 0 - Click first vision-detected conversation")
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

func debugOCR(bridge windows.BridgeInterface, handle uintptr, lang string) {
	fmt.Printf("=== Debug: OCR Text Extraction for Handle: 0x%X (%d) ===\n\n", handle, handle)

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
	fmt.Printf("  Language: %s\n", lang)
	fmt.Printf("  Split Regions Mode: %v\n", *splitRegions)
	fmt.Println()

	// 类型断言以访问 OCR 方法
	winBridge, ok := bridge.(*windows.Bridge)
	if !ok {
		fmt.Printf("ERROR: Failed to cast bridge to *windows.Bridge\n")
		fmt.Printf("Bridge type: %T\n", bridge)
		return
	}

	var ocrResult windows.OCRDebugResult
	var result adapter.Result

	// 根据标志选择OCR方法
	if *splitRegions {
		fmt.Printf("Using split-regions mode (left_sidebar, message_area, input_area)\n")
		ocrResult, result = winBridge.ExtractTextFromWindowRegions(handle, lang)
	} else {
		fmt.Printf("Using full-window OCR mode\n")
		ocrResult, result = winBridge.ExtractTextFromWindow(handle, lang)
	}

	if result.Status != adapter.StatusSuccess {
		fmt.Printf("ERROR: Failed to extract text: %s\n", result.Error)

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
		fmt.Printf("OCR Extraction Diagnostics:\n")
		for _, diag := range result.Diagnostics {
			fmt.Printf("  - %s\n", diag.Message)
			for k, v := range diag.Context {
				fmt.Printf("    %s: %s\n", k, v)
			}
		}
		fmt.Println()
	}

	// 显示 OCR 结果
	fmt.Printf("OCR Results:\n")
	fmt.Printf("  Window Size: %d x %d\n", ocrResult.WindowWidth, ocrResult.WindowHeight)
	fmt.Printf("  Image Size: %d bytes\n", ocrResult.ImageSize)
	fmt.Printf("  Tesseract Path: %s\n", ocrResult.TesseractPath)
	fmt.Printf("  Processing Time: %v\n", ocrResult.ProcessingTime)

	if ocrResult.Error != "" {
		fmt.Printf("  Error: %s\n", ocrResult.Error)
	}
	fmt.Println()

	// 显示提取的文本（全图模式）
	if !*splitRegions && ocrResult.Text != "" {
		fmt.Printf("Extracted Text (%d characters):\n", len(ocrResult.Text))
		fmt.Println("--- BEGIN TEXT ---")
		fmt.Println(ocrResult.Text)
		fmt.Println("--- END TEXT ---")
		fmt.Println()
	}

	// 显示区域文本（分区域模式或后备模式）
	if len(ocrResult.RegionTexts) > 0 {
		fmt.Printf("Region Texts:\n")
		for regionName, regionText := range ocrResult.RegionTexts {
			textLength := len(regionText)
			fmt.Printf("  [%s] (%d chars): ", regionName, textLength)
			if textLength == 0 {
				fmt.Printf("(empty)\n")
			} else if textLength <= 100 {
				fmt.Printf("%s\n", regionText)
			} else {
				fmt.Printf("%s...\n", regionText[:100])
			}
		}
		fmt.Println()
	} else if *splitRegions {
		fmt.Printf("WARNING: No region texts extracted in split-regions mode\n")
		fmt.Println()
	}

	// 显示区域尺寸信息（分区域模式）
	if *splitRegions && ocrResult.WindowWidth > 0 && ocrResult.WindowHeight > 0 {
		fmt.Printf("Region Dimensions (based on window %dx%d):\n", ocrResult.WindowWidth, ocrResult.WindowHeight)
		fmt.Printf("  left_sidebar:    x=0, y=0, width=%d (30%%), height=%d\n",
			ocrResult.WindowWidth*30/100, ocrResult.WindowHeight)
		fmt.Printf("  message_area:    x=%d, y=0, width=%d (70%%), height=%d (70%%)\n",
			ocrResult.WindowWidth*30/100, ocrResult.WindowWidth*70/100, ocrResult.WindowHeight*70/100)
		fmt.Printf("  input_area:      x=%d, y=%d, width=%d (70%%), height=%d (30%%)\n",
			ocrResult.WindowWidth*30/100, ocrResult.WindowHeight*70/100, ocrResult.WindowWidth*70/100, ocrResult.WindowHeight*30/100)
		fmt.Println()
	}

	// JSON 输出
	if *jsonOutput {
		jsonData, err := json.MarshalIndent(ocrResult, "", "  ")
		if err != nil {
			fmt.Printf("ERROR: Failed to marshal OCR result to JSON: %v\n", err)
		} else {
			fmt.Println(string(jsonData))
		}
	}

	fmt.Println("=== Debug OCR Complete ===")
}

func debugVision(bridge windows.BridgeInterface, handle uintptr) {
	fmt.Printf("=== Debug: Vision Detection for Handle: 0x%X (%d) ===\n\n", handle, handle)

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

	// 类型断言以访问视觉检测方法
	winBridge, ok := bridge.(*windows.Bridge)
	if !ok {
		fmt.Printf("ERROR: Failed to cast bridge to *windows.Bridge\n")
		fmt.Printf("Bridge type: %T\n", bridge)
		return
	}

	// 调用视觉检测
	visionResult, result := winBridge.DetectConversations(handle)
	if result.Status != adapter.StatusSuccess {
		fmt.Printf("ERROR: Failed to detect conversations: %s\n", result.Error)

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
		fmt.Printf("Vision Detection Diagnostics:\n")
		for _, diag := range result.Diagnostics {
			fmt.Printf("  - %s\n", diag.Message)
			for k, v := range diag.Context {
				fmt.Printf("    %s: %s\n", k, v)
			}
		}
		fmt.Println()
	}

	// 显示视觉检测结果
	fmt.Printf("Vision Detection Results:\n")
	fmt.Printf("  Window Size: %d x %d\n", visionResult.WindowWidth, visionResult.WindowHeight)
	fmt.Printf("  Image Size: %d bytes\n", visionResult.ImageSize)
	fmt.Printf("  Processing Time: %v\n", visionResult.ProcessingTime)

	if visionResult.Error != "" {
		fmt.Printf("  Error: %s\n", visionResult.Error)
	}
	fmt.Println()

	// 显示左侧会话列表区域
	fmt.Printf("Left Sidebar Region:\n")
	fmt.Printf("  x=%d, y=%d, width=%d, height=%d\n",
		visionResult.LeftSidebarRect[0], visionResult.LeftSidebarRect[1],
		visionResult.LeftSidebarRect[2], visionResult.LeftSidebarRect[3])
	fmt.Println()

	// 显示检测到的会话项
	fmt.Printf("Detected Conversations: %d\n", len(visionResult.ConversationRects))
	if len(visionResult.ConversationRects) > 0 {
		fmt.Println("Conversation Items:")
		for _, conv := range visionResult.ConversationRects {
			fmt.Printf("  [%d] x=%d, y=%d, w=%d, h=%d\n", conv.Index, conv.X, conv.Y, conv.Width, conv.Height)
			fmt.Printf("      Features: ")
			features := []string{}
			if conv.HasAvatar {
				features = append(features, "avatar")
			}
			if conv.HasText {
				features = append(features, "text")
			}
			if conv.HasUnreadDot {
				features = append(features, "unread_dot")
			}
			if conv.IsSelected {
				features = append(features, "selected")
			}
			if len(features) == 0 {
				features = append(features, "none")
			}
			fmt.Printf("%s\n", strings.Join(features, ", "))

			// 显示详细区域信息
			if conv.HasAvatar && conv.AvatarRect[2] > 0 {
				fmt.Printf("      Avatar: x=%d, y=%d, w=%d, h=%d\n",
					conv.AvatarRect[0], conv.AvatarRect[1], conv.AvatarRect[2], conv.AvatarRect[3])
			}
			if conv.HasText && conv.TextRect[2] > 0 {
				fmt.Printf("      Text: x=%d, y=%d, w=%d, h=%d\n",
					conv.TextRect[0], conv.TextRect[1], conv.TextRect[2], conv.TextRect[3])
			}
			if conv.HasUnreadDot && conv.UnreadDotRect[2] > 0 {
				fmt.Printf("      Unread Dot: x=%d, y=%d, w=%d, h=%d\n",
					conv.UnreadDotRect[0], conv.UnreadDotRect[1], conv.UnreadDotRect[2], conv.UnreadDotRect[3])
			}
			fmt.Println()
		}
	} else {
		fmt.Printf("  No conversation items detected\n")
		fmt.Println()
	}

	// 显示检测到的特征统计
	fmt.Printf("Detected Features:\n")
	for feature, count := range visionResult.DetectedFeatures {
		fmt.Printf("  %s: %d\n", feature, count)
	}
	fmt.Println()

	// 显示调试图像信息
	if visionResult.DebugImagePath != "" {
		fmt.Printf("Debug Image Saved:\n")
		fmt.Printf("  Path: %s\n", visionResult.DebugImagePath)
		fmt.Printf("  You can open it with any image viewer\n")
		fmt.Println()
	}

	// JSON 输出
	if *jsonOutput {
		jsonData, err := json.MarshalIndent(visionResult, "", "  ")
		if err != nil {
			fmt.Printf("ERROR: Failed to marshal vision result to JSON: %v\n", err)
		} else {
			fmt.Println(string(jsonData))
		}
	}

	fmt.Println("=== Debug Vision Complete ===")
}

// clickConversation 点击视觉检测到的会话项并验证
// strategy: "avatar_center", "text_center", "rect_center", "left_quarter_center", 或空字符串（使用默认优先级）
func clickConversation(bridge windows.BridgeInterface, handle uintptr, index int, strategy string) {
	fmt.Printf("=== Enhanced Click Conversation: Handle 0x%X (%d), Index %d, Strategy '%s' ===\n\n", handle, handle, index, strategy)

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

	// 类型断言以访问视觉检测方法
	winBridge, ok := bridge.(*windows.Bridge)
	if !ok {
		fmt.Printf("ERROR: Failed to cast bridge to *windows.Bridge\n")
		fmt.Printf("Bridge type: %T\n", bridge)
		return
	}

	// ============================================
	// 步骤1：点击前视觉检测和截图
	// ============================================
	fmt.Printf("--- Step 1: Pre-click Vision Detection & Screenshot ---\n")
	beforeResult, result := winBridge.DetectConversations(handle)
	if result.Status != adapter.StatusSuccess {
		fmt.Printf("ERROR: Failed to detect conversations before click: %s\n", result.Error)
		return
	}

	if index < 0 || index >= len(beforeResult.ConversationRects) {
		fmt.Printf("ERROR: Invalid conversation index %d (total: %d)\n", index, len(beforeResult.ConversationRects))
		return
	}

	fmt.Printf("Pre-click: Detected %d conversation(s)\n", len(beforeResult.ConversationRects))
	preConv := beforeResult.ConversationRects[index]
	fmt.Printf("Target Conversation [%d]:\n", index)
	fmt.Printf("  Position: x=%d, y=%d, w=%d, h=%d\n", preConv.X, preConv.Y, preConv.Width, preConv.Height)
	fmt.Printf("  Features: avatar=%v, text=%v, unread_dot=%v, selected=%v\n",
		preConv.HasAvatar, preConv.HasText, preConv.HasUnreadDot, preConv.IsSelected)
	fmt.Println()

	// 点击前截图 (click_before)
	fmt.Printf("Capturing pre-click screenshot...\n")
	beforeScreenshot, err := winBridge.CaptureWindowScreenshot(handle)
	if err != nil {
		fmt.Printf("WARNING: Failed to capture pre-click screenshot: %v\n", err)
		fmt.Printf("  (Continuing without pixel-level verification)\n")
		beforeScreenshot = nil
	} else {
		fmt.Printf("Pre-click screenshot captured: %dx%d pixels\n", beforeScreenshot.Bounds().Dx(), beforeScreenshot.Bounds().Dy())
	}

	// ============================================
	// 步骤2：计算点击点
	// ============================================
	fmt.Printf("--- Step 2: Calculate Click Point ---\n")
	x, y, clickSource, clickDiag := winBridge.GetConversationClickPoint(beforeResult, index, strategy)
	if clickSource == "invalid_strategy" || clickSource == "strategy_unavailable" {
		fmt.Printf("ERROR: Click strategy failed: %s\n", clickDiag.Message)
		return
	}

	fmt.Printf("Click Point Calculation:\n")
	fmt.Printf("  Coordinates: x=%d, y=%d\n", x, y)
	fmt.Printf("  Source: %s\n", clickSource)
	fmt.Printf("  Message: %s\n", clickDiag.Message)
	for k, v := range clickDiag.Context {
		fmt.Printf("  %s: %s\n", k, v)
	}
	fmt.Println()

	// ============================================
	// 步骤3：执行点击
	// ============================================
	fmt.Printf("--- Step 3: Execute Click ---\n")
	clickResult := winBridge.Click(handle, x, y)
	if clickResult.Status != adapter.StatusSuccess {
		fmt.Printf("ERROR: Click failed: %s\n", clickResult.Error)
		return
	}
	fmt.Printf("Click executed successfully\n")

	// ============================================
	// 步骤4：多时刻截图
	// ============================================
	fmt.Printf("--- Step 4: Multi-time Screenshots ---\n")

	type TimedScreenshot struct {
		TimeLabel string
		WaitTime  time.Duration
		Image     *image.RGBA
		DiffResult windows.ImageDifferenceResult
	}

	timePoints := []TimedScreenshot{
		{"click_before", 0, beforeScreenshot, windows.ImageDifferenceResult{}},
	}

	// 定义等待时间点
	waitTimes := []struct {
		label string
		delay time.Duration
	}{
		{"click_after_300ms", 300 * time.Millisecond},
		{"click_after_800ms", 800 * time.Millisecond},
		{"click_after_1500ms", 1500 * time.Millisecond},
	}

	// 获取左侧边栏区域用于左右侧分析
	leftSidebarRect := beforeResult.LeftSidebarRect
	if leftSidebarRect[2] == 0 || leftSidebarRect[3] == 0 {
		// 如果没有检测到左侧边栏，使用默认值（假设左侧30%）
		width := beforeResult.WindowWidth
		if width <= 0 && beforeScreenshot != nil {
			width = beforeScreenshot.Bounds().Dx()
		}
		leftSidebarRect = [4]int{0, 0, width * 30 / 100, beforeResult.WindowHeight}
	}

	// 捕获每个时间点的截图
	for _, wt := range waitTimes {
		fmt.Printf("  Waiting %s...\n", wt.label)
		time.Sleep(wt.delay)

		screenshot, err := winBridge.CaptureWindowScreenshot(handle)
		if err != nil {
			fmt.Printf("  WARNING: Failed to capture %s screenshot: %v\n", wt.label, err)
			continue
		}

		fmt.Printf("  Captured %s: %dx%d pixels\n", wt.label, screenshot.Bounds().Dx(), screenshot.Bounds().Dy())

		// 计算与点击前图像的差异
		var diffResult windows.ImageDifferenceResult
		if beforeScreenshot != nil {
			diffResult, err = windows.ComputeImageDifference(beforeScreenshot, screenshot, leftSidebarRect, beforeResult.WindowWidth)
			if err != nil {
				fmt.Printf("  WARNING: Failed to compute difference for %s: %v\n", wt.label, err)
			} else {
				fmt.Printf("  Difference: %.2f%% pixels changed\n", diffResult.DifferencePercent)
			}
		}

		timePoints = append(timePoints, TimedScreenshot{
			TimeLabel: wt.label,
			WaitTime:  wt.delay,
			Image:     screenshot,
			DiffResult: diffResult,
		})
	}

	// ============================================
	// 步骤5：增强验证（4种验证信号）
	// ============================================
	fmt.Printf("--- Step 5: Enhanced Verification ---\n")

	verificationSignals := make(map[string]bool)
	signalDetails := make(map[string]string)

	// 使用最后一次截图进行验证（1500ms后）
	var lastDiff windows.ImageDifferenceResult
	if len(timePoints) > 1 {
		lastDiff = timePoints[len(timePoints)-1].DiffResult
	}

	// 信号1：左侧被点击会话项区域的像素差异
	if preConv.X >= 0 && preConv.Y >= 0 && preConv.Width > 0 && preConv.Height > 0 {
		// 定义会话项区域（扩大一些以捕获周围变化）
		regionX := preConv.X - 5
		regionY := preConv.Y - 5
		regionWidth := preConv.Width + 10
		regionHeight := preConv.Height + 10

		if regionX < 0 { regionX = 0 }
		if regionY < 0 { regionY = 0 }

		if beforeScreenshot != nil && timePoints[len(timePoints)-1].Image != nil {
			convDiffCount, convDiffPercent, err := windows.ComputeRegionDifference(
				beforeScreenshot,
				timePoints[len(timePoints)-1].Image,
				regionX, regionY, regionWidth, regionHeight,
			)

			if err == nil {
				verificationSignals["clicked_region_pixel_diff"] = convDiffPercent > 0.5 // 阈值0.5%
				signalDetails["clicked_region_pixel_diff"] = fmt.Sprintf("%.2f%% (count=%d)", convDiffPercent, convDiffCount)
				fmt.Printf("✓ Clicked region pixel diff: %.2f%% (%d pixels)\n", convDiffPercent, convDiffCount)
			}
		}
	}

	// 信号2：右侧消息区的像素差异
	if beforeScreenshot != nil && timePoints[len(timePoints)-1].Image != nil {
		rightRegionX := leftSidebarRect[0] + leftSidebarRect[2]
		rightRegionWidth := beforeResult.WindowWidth - rightRegionX
		if rightRegionWidth > 0 {
			rightDiffCount, rightDiffPercent, err := windows.ComputeRegionDifference(
				beforeScreenshot,
				timePoints[len(timePoints)-1].Image,
				rightRegionX, 0, rightRegionWidth, beforeResult.WindowHeight,
			)

			if err == nil {
				verificationSignals["right_region_pixel_diff"] = rightDiffPercent > 0.5
				signalDetails["right_region_pixel_diff"] = fmt.Sprintf("%.2f%% (count=%d)", rightDiffPercent, rightDiffCount)
				fmt.Printf("✓ Right region pixel diff: %.2f%% (%d pixels)\n", rightDiffPercent, rightDiffCount)
			}
		}
	}

	// 信号3：整窗截图差异面积
	if lastDiff.TotalPixels > 0 {
		verificationSignals["whole_window_diff_area"] = lastDiff.DifferencePercent > 0.2
		signalDetails["whole_window_diff_area"] = fmt.Sprintf("%.2f%% (%d pixels)", lastDiff.DifferencePercent, lastDiff.DifferentPixels)
		fmt.Printf("✓ Whole window diff area: %.2f%% (%d/%d pixels)\n",
			lastDiff.DifferencePercent, lastDiff.DifferentPixels, lastDiff.TotalPixels)
	}

	// 信号4：差异热区的 bounding box
	if lastDiff.DifferentPixels > 0 {
		verificationSignals["diff_bounding_box"] = lastDiff.DiffBoundingBox[2] > 10 && lastDiff.DiffBoundingBox[3] > 10
		signalDetails["diff_bounding_box"] = fmt.Sprintf("x=%d,y=%d,w=%d,h=%d",
			lastDiff.DiffBoundingBox[0], lastDiff.DiffBoundingBox[1],
			lastDiff.DiffBoundingBox[2], lastDiff.DiffBoundingBox[3])
		fmt.Printf("✓ Diff bounding box: x=%d,y=%d,w=%d,h=%d\n",
			lastDiff.DiffBoundingBox[0], lastDiff.DiffBoundingBox[1],
			lastDiff.DiffBoundingBox[2], lastDiff.DiffBoundingBox[3])
	}

	// 左右侧差异分析
	if lastDiff.TotalPixels > 0 {
		fmt.Printf("✓ Left/Right analysis:\n")
		fmt.Printf("  Left side: %.2f%% (%d pixels)\n", lastDiff.LeftSidePercent, lastDiff.LeftSideDiffPixels)
		fmt.Printf("  Right side: %.2f%% (%d pixels)\n", lastDiff.RightSidePercent, lastDiff.RightSideDiffPixels)

		if lastDiff.LeftSidePercent > lastDiff.RightSidePercent * 2 {
			signalDetails["change_location"] = "predominantly_left"
			fmt.Printf("  Change predominantly in left side (%.1fx more)\n",
				lastDiff.LeftSidePercent / max(lastDiff.RightSidePercent, 0.01))
		} else if lastDiff.RightSidePercent > lastDiff.LeftSidePercent * 2 {
			signalDetails["change_location"] = "predominantly_right"
			fmt.Printf("  Change predominantly in right side (%.1fx more)\n",
				lastDiff.RightSidePercent / max(lastDiff.LeftSidePercent, 0.01))
		} else {
			signalDetails["change_location"] = "balanced"
			fmt.Printf("  Change balanced between left and right\n")
		}
	}

	// 传统的视觉检测验证（作为补充）
	fmt.Printf("\n--- Step 6: Traditional Vision Detection Verification ---\n")
	afterResult, result := winBridge.DetectConversations(handle)
	if result.Status == adapter.StatusSuccess {
		if len(afterResult.ConversationRects) > index {
			postConv := afterResult.ConversationRects[index]

			// 选中状态变化
			if preConv.IsSelected != postConv.IsSelected {
				verificationSignals["selection_state_changed"] = true
				signalDetails["selection_state_changed"] = fmt.Sprintf("%v->%v", preConv.IsSelected, postConv.IsSelected)
				fmt.Printf("✓ Selection state changed: %v -> %v\n", preConv.IsSelected, postConv.IsSelected)
			} else {
				fmt.Printf("- Selection state unchanged: %v\n", preConv.IsSelected)
			}
		}
	}

	// ============================================
	// 步骤7：总结和评估
	// ============================================
	fmt.Printf("\n--- Step 7: Summary & Evaluation ---\n")

	// 计算验证信号通过数
	passedSignals := 0
	for _, passed := range verificationSignals {
		if passed {
			passedSignals++
		}
	}

	totalSignals := len(verificationSignals)
	fmt.Printf("Verification Signals: %d/%d passed\n", passedSignals, totalSignals)

	// 列出所有信号状态
	for signal, passed := range verificationSignals {
		status := "FAIL"
		if passed { status = "PASS" }
		fmt.Printf("  %-30s: %s (%s)\n", signal, status, signalDetails[signal])
	}

	// 多时刻差异分析
	fmt.Printf("\nMulti-time Difference Analysis (vs click_before):\n")
	fmt.Printf("%-20s %-12s %-12s %-12s\n", "Time Point", "Diff %", "Left %", "Right %")
	fmt.Printf("%-20s %-12s %-12s %-12s\n", "--------------------", "------------", "------------", "------------")

	for _, tp := range timePoints[1:] { // 跳过click_before
		if tp.DiffResult.TotalPixels > 0 {
			fmt.Printf("%-20s %-12.2f %-12.2f %-12.2f\n",
				tp.TimeLabel,
				tp.DiffResult.DifferencePercent,
				tp.DiffResult.LeftSidePercent,
				tp.DiffResult.RightSidePercent)
		}
	}

	// 总体评估
	fmt.Printf("\nOverall Assessment:\n")
	if passedSignals >= 2 {
		fmt.Printf("✓ STRONG INDICATION: Click likely hit the target conversation item\n")
		fmt.Printf("  Multiple verification signals detected significant interface changes\n")
	} else if passedSignals == 1 {
		fmt.Printf("○ WEAK INDICATION: Click may have hit the target\n")
		fmt.Printf("  Only one verification signal detected, changes may be subtle\n")
	} else {
		fmt.Printf("✗ NO CLEAR INDICATION: Click may have missed or had no effect\n")
		fmt.Printf("  No verification signals detected, possible reasons:\n")
		fmt.Printf("  - Click hit wrong area\n")
		fmt.Printf("  - Interface changes are too subtle for pixel diff\n")
		fmt.Printf("  - Item was already selected\n")
		fmt.Printf("  - Verification thresholds too high\n")
	}

	// 点击策略评估
	fmt.Printf("\nClick Strategy Analysis:\n")
	fmt.Printf("  Strategy used: %s\n", clickSource)
	fmt.Printf("  Coordinates: (%d, %d)\n", x, y)
	fmt.Printf("  Recommended for next test: ")

	if verificationSignals["clicked_region_pixel_diff"] && verificationSignals["selection_state_changed"] {
		fmt.Printf("Current strategy (%s) works well\n", clickSource)
	} else if signalDetails["change_location"] == "predominantly_left" {
		fmt.Printf("Try avatar_center or left_quarter_center\n")
	} else {
		fmt.Printf("Try different strategy (avatar_center, text_center, etc.)\n")
	}

	fmt.Printf("\n=== Enhanced Click Conversation Complete ===\n")
}

// focusVision 视觉Focus统一入口
func focusVision(bridge windows.BridgeInterface, handle uintptr, index int, strategy string, waitMs int) {
	fmt.Printf("=== Vision Focus: Handle 0x%X (%d), Index %d, Strategy '%s', Wait %dms ===\n\n", handle, handle, index, strategy, waitMs)

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

	// 类型断言以访问视觉检测方法
	winBridge, ok := bridge.(*windows.Bridge)
	if !ok {
		fmt.Printf("ERROR: Failed to cast bridge to *windows.Bridge\n")
		fmt.Printf("Bridge type: %T\n", bridge)
		return
	}

	// 调用视觉Focus统一入口
	fmt.Printf("--- Executing Vision Focus ---\n")
	focusResult, result := winBridge.FocusConversationByVision(handle, strategy, index, waitMs)
	if result.Status != adapter.StatusSuccess {
		fmt.Printf("ERROR: Vision focus failed: %s\n", result.Error)
		if focusResult.Error != "" {
			fmt.Printf("  Additional error: %s\n", focusResult.Error)
		}
		return
	}

	// ============================================
	// 显示Focus结果摘要
	// ============================================
	fmt.Printf("\n=== Vision Focus Results ===\n")

	// 目标项信息
	fmt.Printf("Target Selection:\n")
	fmt.Printf("  Index: %d (selection source: %v)\n",
		focusResult.TargetIndex,
		focusResult.VerificationSignals["selection_source"])
	fmt.Printf("  Position: x=%d, y=%d, w=%d, h=%d\n",
		focusResult.TargetRect.X, focusResult.TargetRect.Y,
		focusResult.TargetRect.Width, focusResult.TargetRect.Height)
	fmt.Printf("  Features: avatar=%v, text=%v, unread_dot=%v, selected=%v\n",
		focusResult.TargetRect.HasAvatar, focusResult.TargetRect.HasText,
		focusResult.TargetRect.HasUnreadDot, focusResult.TargetRect.IsSelected)

	// 点击信息
	fmt.Printf("\nClick Execution:\n")
	fmt.Printf("  Strategy: %s\n", focusResult.ClickStrategy)
	fmt.Printf("  Source: %s\n", focusResult.ClickSource)
	fmt.Printf("  Coordinates: x=%d, y=%d\n", focusResult.ClickX, focusResult.ClickY)

	// Focus验证结果
	fmt.Printf("\nFocus Verification:\n")
	fmt.Printf("  Success: %v\n", focusResult.FocusSucceeded)
	fmt.Printf("  Confidence: %.2f\n", focusResult.FocusConfidence)
	if len(focusResult.SuccessReasons) > 0 {
		fmt.Printf("  Reasons: %s\n", strings.Join(focusResult.SuccessReasons, ", "))
	} else {
		fmt.Printf("  Reasons: (none)\n")
	}

	// 详细验证信号
	fmt.Printf("\nDetailed Verification Signals:\n")
	for key, value := range focusResult.VerificationSignals {
		// 跳过click_diagnostic（它是Diagnostic对象，打印会有问题）
		if key == "click_diagnostic" {
			continue
		}
		fmt.Printf("  %s: %v\n", key, value)
	}

	// 显示诊断信息
	fmt.Printf("\nDiagnostics:\n")
	for _, diag := range result.Diagnostics {
		fmt.Printf("  [%s] %s\n", diag.Level, diag.Message)
		for k, v := range diag.Context {
			fmt.Printf("    %s: %s\n", k, v)
		}
	}

	// 调试图像路径
	if focusResult.DebugImagePath != "" {
		fmt.Printf("\nDebug Resources:\n")
		fmt.Printf("  Debug image: %s\n", focusResult.DebugImagePath)
	}

	// 总体评估
	fmt.Printf("\n=== Overall Assessment ===\n")
	if focusResult.FocusSucceeded {
		fmt.Printf("✓ FOCUS SUCCESS: Conversation focus achieved with confidence %.2f\n", focusResult.FocusConfidence)
		if focusResult.FocusConfidence >= 0.8 {
			fmt.Printf("  High confidence: Visual verification strongly indicates successful focus\n")
		} else if focusResult.FocusConfidence >= 0.5 {
			fmt.Printf("  Medium confidence: Multiple verification signals detected\n")
		} else {
			fmt.Printf("  Low confidence: Some verification signals detected, but confidence is low\n")
		}
	} else {
		fmt.Printf("✗ FOCUS FAILED: Unable to confirm successful focus\n")
		fmt.Printf("  Confidence: %.2f\n", focusResult.FocusConfidence)
		fmt.Printf("  Possible reasons:\n")
		fmt.Printf("  - Click missed target conversation\n")
		fmt.Printf("  - Interface changes too subtle for pixel diff\n")
		fmt.Printf("  - Item already selected\n")
		fmt.Printf("  - Verification thresholds too strict\n")
	}

	// 建议
	fmt.Printf("\n=== Recommendations ===\n")
	if focusResult.FocusSucceeded {
		fmt.Printf("  Visual Focus prototype is WORKING\n")
		fmt.Printf("  Next step: Integrate visual focus into adapter/wechat Focus() method\n")
	} else {
		fmt.Printf("  Check detection quality first: bridge-dump debug-vision %d\n", handle)
		fmt.Printf("  Try different click strategy: %s\n",
			getRecommendedStrategy(focusResult.ClickStrategy, focusResult.FocusConfidence))
		fmt.Printf("  Consider adjusting verification thresholds if needed\n")
	}

	fmt.Printf("\n=== Vision Focus Complete ===\n")
	fmt.Printf("Processing time: %v\n", focusResult.ProcessingTime)
}

// getRecommendedStrategy 根据当前策略和置信度推荐下一个策略
func getRecommendedStrategy(currentStrategy string, confidence float64) string {
	strategies := []string{"rect_center", "left_quarter_center", "text_center", "avatar_center"}

	// 如果置信度低，推荐其他策略
	if confidence < 0.5 {
		for _, s := range strategies {
			if s != currentStrategy {
				return s
			}
		}
	}

	// 否则推荐当前策略或默认
	if currentStrategy == "" {
		return "rect_center"
	}
	return currentStrategy
}

// max 辅助函数
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// abs 绝对值辅助函数
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// debugInputBox 调试输入框检测
func debugInputBox(bridge windows.BridgeInterface, handle uintptr) {
	fmt.Printf("=== Debug: Input Box Detection for Handle: 0x%X (%d) ===\n\n", handle, handle)

	// 获取窗口信息
	info, infoResult := bridge.GetWindowInfo(handle)
	if infoResult.Status != adapter.StatusSuccess {
		fmt.Printf("Failed to get window info: %s\n", infoResult.Error)
		return
	}
	fmt.Printf("Window Info: Handle=0x%X, Class=%s, Title=%s\n\n", info.Handle, info.Class, info.Title)

	// 检测会话列表（获取左侧边栏矩形）
	visionResult, visionDetectResult := bridge.DetectConversations(handle)
	if visionDetectResult.Status != adapter.StatusSuccess {
		fmt.Printf("Failed to detect conversations: %s\n", visionDetectResult.Error)
		return
	}

	fmt.Printf("Vision Detection Results:\n")
	fmt.Printf("  Window Size: %dx%d\n", visionResult.WindowWidth, visionResult.WindowHeight)
	fmt.Printf("  Left Sidebar Rect: [%d,%d,%d,%d]\n",
		visionResult.LeftSidebarRect[0], visionResult.LeftSidebarRect[1],
		visionResult.LeftSidebarRect[2], visionResult.LeftSidebarRect[3])
	fmt.Printf("  Conversation Rects: %d\n", len(visionResult.ConversationRects))
	if visionResult.DebugImagePath != "" {
		fmt.Printf("  Debug Image: %s\n", visionResult.DebugImagePath)
	}
	fmt.Println()

	// 检测输入框区域
	inputBoxRect, inputBoxResult := bridge.DetectInputBoxArea(
		handle,
		visionResult.LeftSidebarRect,
		visionResult.WindowWidth,
		visionResult.WindowHeight,
	)

	fmt.Printf("Input Box Detection Results:\n")
	fmt.Printf("  Input Box Rect: X=%d, Y=%d, Width=%d, Height=%d\n",
		inputBoxRect.X, inputBoxRect.Y, inputBoxRect.Width, inputBoxRect.Height)

	// 输出诊断信息
	for _, diag := range inputBoxResult.Diagnostics {
		fmt.Printf("  Detection Method: %s\n", diag.Context["detection_method"])
		fmt.Printf("  Detection Score: %s\n", diag.Context["detection_score"])
		if debugPath := diag.Context["debug_image_path"]; debugPath != "" {
			fmt.Printf("  Debug Image: %s\n", debugPath)
		}
	}

	// 计算不同策略的点击坐标
	fmt.Printf("\nClick Points by Strategy:\n")
	strategies := []string{"input_left_third", "input_center", "input_left_quarter", "input_double_click_center"}
	for _, strategy := range strategies {
		clickX, clickY, clickSource := bridge.GetInputBoxClickPoint(inputBoxRect, strategy)
		fmt.Printf("  %-25s: (%d, %d) [%s]\n", strategy, clickX, clickY, clickSource)
	}

	fmt.Println("\n=== Input Box Debug Complete ===")
}

// clickInputBox 点击输入框并验证
func clickInputBox(bridge windows.BridgeInterface, handle uintptr, strategy string) {
	fmt.Printf("=== Click Input Box: Handle=0x%X, Strategy=%s ===\n\n", handle, strategy)

	// 检测会话列表
	visionResult, visionDetectResult := bridge.DetectConversations(handle)
	if visionDetectResult.Status != adapter.StatusSuccess {
		fmt.Printf("Failed to detect conversations: %s\n", visionDetectResult.Error)
		return
	}

	// 检测输入框区域
	inputBoxRect, inputBoxResult := bridge.DetectInputBoxArea(
		handle,
		visionResult.LeftSidebarRect,
		visionResult.WindowWidth,
		visionResult.WindowHeight,
	)

	fmt.Printf("Input Box: X=%d, Y=%d, Width=%d, Height=%d\n",
		inputBoxRect.X, inputBoxRect.Y, inputBoxRect.Width, inputBoxRect.Height)

	// 输出检测方法信息
	if len(inputBoxResult.Diagnostics) > 0 {
		for _, diag := range inputBoxResult.Diagnostics {
			if method := diag.Context["detection_method"]; method != "" {
				fmt.Printf("Detection Method: %s\n", method)
			}
			if score := diag.Context["detection_score"]; score != "" {
				fmt.Printf("Detection Score: %s\n", score)
			}
			if debugPath := diag.Context["debug_image_path"]; debugPath != "" {
				fmt.Printf("Debug Image: %s\n", debugPath)
			}
		}
	}

	// 计算点击坐标
	clickX, clickY, clickSource := bridge.GetInputBoxClickPoint(inputBoxRect, strategy)
	fmt.Printf("Click Point: (%d, %d) [%s]\n", clickX, clickY, clickSource)

	// 捕获点击前截图
	fmt.Printf("Capturing before-click screenshot...\n")
	beforeScreenshot, captureResult := bridge.CaptureWindow(handle)
	if captureResult.Status != adapter.StatusSuccess {
		fmt.Printf("Failed to capture before screenshot: %s\n", captureResult.Error)
		return
	}

	// 点击输入框
	fmt.Printf("Clicking input box...\n")
	clickResult := bridge.Click(handle, clickX, clickY)
	if clickResult.Status != adapter.StatusSuccess {
		fmt.Printf("Click failed: %s\n", clickResult.Error)
		return
	}
	fmt.Printf("Click successful\n")

	// 等待点击生效
	time.Sleep(200 * time.Millisecond)

	// 捕获点击后截图
	fmt.Printf("Capturing after-click screenshot...\n")
	afterScreenshot, captureResult := bridge.CaptureWindow(handle)
	if captureResult.Status != adapter.StatusSuccess {
		fmt.Printf("Failed to capture after screenshot: %s\n", captureResult.Error)
		return
	}

	// 计算输入框区域差异
	if len(beforeScreenshot) > 0 && len(afterScreenshot) > 0 {
		diff := windows.CalculateRectDiffPercent(
			beforeScreenshot, afterScreenshot,
			visionResult.WindowWidth, visionResult.WindowHeight,
			inputBoxRect,
		)
		fmt.Printf("Input Box Diff After Click: %.3f\n", diff)
		if diff > 0 {
			fmt.Printf("✓ Input box activated (diff > 0)\n")
		} else {
			fmt.Printf("✗ Input box NOT activated (diff = 0)\n")
		}
	} else {
		fmt.Printf("Cannot calculate diff: screenshots empty\n")
	}

	fmt.Println("\n=== Click Input Box Complete ===")
}


