package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strconv"

	"github.com/yourorg/auto-customer-service/internal/agent/adapter"
	"github.com/yourorg/auto-customer-service/internal/agent/windows"
)

var (
	jsonOutput = flag.Bool("json", false, "Output as JSON")
	maxDepth   = flag.Int("depth", 5, "Maximum recursion depth for node traversal")
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
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --json                              - Output as JSON")
	fmt.Println("  --depth <n>                         - Maximum recursion depth (default: 5)")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  bridge-dump find-wechat")
	fmt.Println("  bridge-dump window-info 123456")
	fmt.Println("  bridge-dump list-nodes 123456 --json --depth 3")
	fmt.Println("  bridge-dump focus 123456")
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
		printNodesText(nodes, 0)
	}
}

func printNodesText(nodes []windows.AccessibleNode, depth int) {
	if depth >= *maxDepth {
		return
	}

	indent := ""
	for i := 0; i < depth; i++ {
		indent += "  "
	}

	for i, node := range nodes {
		fmt.Printf("%s[%d] Handle: %d, Name: %s, Role: %s, Class: %s\n",
			indent, i+1, node.Handle, node.Name, node.Role, node.ClassName)

		if len(node.Children) > 0 {
			printNodesText(node.Children, depth+1)
		}
	}
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
