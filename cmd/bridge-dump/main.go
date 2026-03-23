package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/yourorg/auto-customer-service/internal/agent/adapter"
	"github.com/yourorg/auto-customer-service/internal/agent/windows"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]
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
		if len(os.Args) < 3 {
			log.Fatal("Usage: bridge-dump window-info <window-handle>")
		}
		handle, err := strconv.ParseUint(os.Args[2], 10, 64)
		if err != nil {
			log.Fatalf("Invalid window handle: %v", err)
		}
		windowInfo(bridge, uintptr(handle))
	case "list-nodes":
		if len(os.Args) < 3 {
			log.Fatal("Usage: bridge-dump list-nodes <window-handle>")
		}
		handle, err := strconv.ParseUint(os.Args[2], 10, 64)
		if err != nil {
			log.Fatalf("Invalid window handle: %v", err)
		}
		listNodes(bridge, uintptr(handle))
	case "focus":
		if len(os.Args) < 3 {
			log.Fatal("Usage: bridge-dump focus <window-handle>")
		}
		handle, err := strconv.ParseUint(os.Args[2], 10, 64)
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
	fmt.Println("Examples:")
	fmt.Println("  bridge-dump find-wechat")
	fmt.Println("  bridge-dump window-info 123456")
	fmt.Println("  bridge-dump list-nodes 123456")
	fmt.Println("  bridge-dump focus 123456")
}

func findWeChat(bridge windows.BridgeInterface) {
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

func windowInfo(bridge windows.BridgeInterface, handle uintptr) {
	info, result := bridge.GetWindowInfo(handle)
	if result.Status != adapter.StatusSuccess {
		log.Fatalf("Failed to get window info: %s", result.Error)
	}

	// Output as JSON for easy parsing
	data := map[string]interface{}{
		"handle": handle,
		"class":  info.Class,
		"title":  info.Title,
	}

	jsonData, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(jsonData))
}

func listNodes(bridge windows.BridgeInterface, handle uintptr) {
	nodes, result := bridge.EnumerateAccessibleNodes(handle)
	if result.Status != adapter.StatusSuccess {
		log.Fatalf("Failed to enumerate nodes: %s", result.Error)
	}

	fmt.Printf("Found %d accessibility node(s):\n", len(nodes))
	for i, node := range nodes {
		fmt.Printf("  [%d] Handle: %d, Name: %s, Role: %s, Class: %s\n",
			i+1, node.Handle, node.Name, node.Role, node.ClassName)
	}
}

func focusWindow(bridge windows.BridgeInterface, handle uintptr) {
	result := bridge.FocusWindow(handle)
	if result.Status != adapter.StatusSuccess {
		log.Fatalf("Failed to focus window: %s", result.Error)
	}

	fmt.Printf("Successfully focused window: %d\n", handle)
}
