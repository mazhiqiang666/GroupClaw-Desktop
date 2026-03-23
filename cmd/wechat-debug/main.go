package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/yourorg/auto-customer-service/internal/agent/adapter"
	"github.com/yourorg/auto-customer-service/internal/agent/adapter/wechat"
	"github.com/yourorg/auto-customer-service/pkg/protocol"
)

var (
	jsonOutput = flag.Bool("json", false, "Output as JSON")
	contactName = flag.String("contact", "", "Contact name to focus/send to")
	messageContent = flag.String("message", "Test message from debugging script", "Message content to send")
)

func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
		printUsage()
		return
	}

	command := args[0]

	// 创建 WeChat 适配器
	wechatAdapter := wechat.NewWeChatAdapter()

	// 初始化适配器
	result := wechatAdapter.Init(adapter.Config{})
	if result.Status != adapter.StatusSuccess {
		log.Fatalf("Failed to initialize adapter: %s", result.Error)
	}
	defer wechatAdapter.Destroy()

	switch command {
	case "find-window":
		findWeChatWindow(wechatAdapter)
	case "list-nodes":
		if len(args) < 2 {
			log.Fatal("Usage: wechat-debug list-nodes <window-handle>")
		}
		handle, err := strconv.ParseUint(args[1], 10, 64)
		if err != nil {
			log.Fatalf("Invalid window handle: %v", err)
		}
		listNodes(wechatAdapter, uintptr(handle))
	case "scan":
		scanConversations(wechatAdapter)
	case "focus":
		if *contactName == "" {
			log.Fatal("Usage: wechat-debug focus --contact <name>")
		}
		focusContact(wechatAdapter, *contactName)
	case "send":
		if *contactName == "" {
			log.Fatal("Usage: wechat-debug send --contact <name> [--message <content>]")
		}
		sendMessage(wechatAdapter, *contactName, *messageContent)
	case "verify":
		if *contactName == "" {
			log.Fatal("Usage: wechat-debug verify --contact <name> [--message <content>]")
		}
		verifyMessage(wechatAdapter, *contactName, *messageContent)
	case "full-test":
		fullTestFlow(wechatAdapter)
	case "run-chain":
		runChain(wechatAdapter)
	default:
		printUsage()
	}
}

func printUsage() {
	fmt.Println("wechat-debug - WeChat debugging and testing tool")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  wechat-debug find-window              - Find WeChat window(s)")
	fmt.Println("  wechat-debug list-nodes <handle>      - List accessibility nodes")
	fmt.Println("  wechat-debug scan                     - Scan conversation list")
	fmt.Println("  wechat-debug focus --contact <name>   - Focus on specific contact")
	fmt.Println("  wechat-debug send --contact <name> [--message <content>] - Send message to contact")
	fmt.Println("  wechat-debug verify --contact <name> [--message <content>] - Verify message delivery")
	fmt.Println("  wechat-debug full-test                - Run full test flow")
	fmt.Println("  wechat-debug run-chain                - Run complete chain: scan -> focus -> send -> verify")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --json                                - Output as JSON")
	fmt.Println("  --contact <name>                      - Contact name for focus/send/verify")
	fmt.Println("  --message <content>                   - Message content to send/verify")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  wechat-debug find-window")
	fmt.Println("  wechat-debug list-nodes 123456")
	fmt.Println("  wechat-debug scan")
	fmt.Println("  wechat-debug focus --contact \"张三\"")
	fmt.Println("  wechat-debug send --contact \"张三\" --message \"Hello from debug script\"")
	fmt.Println("  wechat-debug verify --contact \"张三\" --message \"Hello from debug script\"")
	fmt.Println("  wechat-debug full-test")
	fmt.Println("  wechat-debug run-chain --contact \"张三\" --message \"Test message\"")
}

func findWeChatWindow(wechatAdapter *wechat.WeChatAdapter) {
	instances, result := wechatAdapter.Detect()
	if result.Status != adapter.StatusSuccess {
		log.Fatalf("Failed to detect WeChat: %s", result.Error)
	}

	if len(instances) == 0 {
		fmt.Println("No WeChat windows found")
		return
	}

	if *jsonOutput {
		data := map[string]interface{}{
			"instances": instances,
			"count":     len(instances),
		}
		jsonData, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(jsonData))
	} else {
		fmt.Printf("Found %d WeChat instance(s):\n", len(instances))
		for i, inst := range instances {
			fmt.Printf("  [%d] AppID: %s, InstanceID: %s\n", i+1, inst.AppID, inst.InstanceID)
		}
	}
}

func listNodes(wechatAdapter *wechat.WeChatAdapter, windowHandle uintptr) {
	// Note: This requires access to the bridge, which is internal
	// For now, we'll use the adapter's Scan method to get node info
	instances, result := wechatAdapter.Detect()
	if result.Status != adapter.StatusSuccess || len(instances) == 0 {
		log.Fatal("No WeChat window found")
	}

	// Use scan to get node information
	conversations, scanResult := wechatAdapter.Scan(instances[0])
	if scanResult.Status != adapter.StatusSuccess {
		log.Fatalf("Failed to scan: %s", scanResult.Error)
	}

	if *jsonOutput {
		data := map[string]interface{}{
			"window_handle":   windowHandle,
			"conversations":   conversations,
			"conversation_count": len(conversations),
		}
		jsonData, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(jsonData))
	} else {
		fmt.Printf("Found %d conversation(s):\n", len(conversations))
		for i, conv := range conversations {
			fmt.Printf("  [%d] %s (position: %d)\n", i+1, conv.DisplayName, conv.ListPosition)
			if len(conv.ListNeighborhoodHint) > 0 {
				fmt.Printf("      Path: %s\n", conv.ListNeighborhoodHint[0])
			}
		}
	}
}

func scanConversations(wechatAdapter *wechat.WeChatAdapter) {
	instances, result := wechatAdapter.Detect()
	if result.Status != adapter.StatusSuccess || len(instances) == 0 {
		log.Fatal("No WeChat window found")
	}

	conversations, scanResult := wechatAdapter.Scan(instances[0])
	if scanResult.Status != adapter.StatusSuccess {
		log.Fatalf("Failed to scan: %s", scanResult.Error)
	}

	if *jsonOutput {
		data := map[string]interface{}{
			"conversations":   conversations,
			"conversation_count": len(conversations),
			"diagnostics":     scanResult.Diagnostics,
		}
		jsonData, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(jsonData))
	} else {
		fmt.Printf("Scan completed:\n")
		fmt.Printf("  Conversations found: %d\n", len(conversations))
		for i, conv := range conversations {
			fmt.Printf("  [%d] %s (position: %d)\n", i+1, conv.DisplayName, conv.ListPosition)
		}
		fmt.Printf("\nDiagnostics:\n")
		for _, diag := range scanResult.Diagnostics {
			fmt.Printf("  %s: %s\n", diag.Level, diag.Message)
			for k, v := range diag.Context {
				fmt.Printf("    %s: %s\n", k, v)
			}
		}
	}
}

func focusContact(wechatAdapter *wechat.WeChatAdapter, contactName string) {
	instances, result := wechatAdapter.Detect()
	if result.Status != adapter.StatusSuccess || len(instances) == 0 {
		log.Fatal("No WeChat window found")
	}

	conversations, scanResult := wechatAdapter.Scan(instances[0])
	if scanResult.Status != adapter.StatusSuccess {
		log.Fatalf("Failed to scan: %s", scanResult.Error)
	}

	// Find the contact
	var targetConv *protocol.ConversationRef
	for i := range conversations {
		if conversations[i].DisplayName == contactName {
			targetConv = &conversations[i]
			break
		}
	}

	if targetConv == nil {
		log.Fatalf("Contact '%s' not found in conversation list", contactName)
	}

	fmt.Printf("Focusing on contact: %s\n", contactName)

	focusResult := wechatAdapter.Focus(*targetConv)

	if *jsonOutput {
		data := map[string]interface{}{
			"contact":    contactName,
			"success":    focusResult.Status == adapter.StatusSuccess,
			"confidence": focusResult.Confidence,
			"diagnostics": focusResult.Diagnostics,
		}
		jsonData, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(jsonData))
	} else {
		fmt.Printf("Focus result:\n")
		fmt.Printf("  Status: %s\n", focusResult.Status)
		fmt.Printf("  Confidence: %.2f\n", focusResult.Confidence)
		fmt.Printf("  Diagnostics:\n")
		for _, diag := range focusResult.Diagnostics {
			fmt.Printf("    %s: %s\n", diag.Level, diag.Message)
			for k, v := range diag.Context {
				fmt.Printf("      %s: %s\n", k, v)
			}
		}
	}
}

func sendMessage(wechatAdapter *wechat.WeChatAdapter, contactName string, message string) {
	instances, result := wechatAdapter.Detect()
	if result.Status != adapter.StatusSuccess || len(instances) == 0 {
		log.Fatal("No WeChat window found")
	}

	conversations, scanResult := wechatAdapter.Scan(instances[0])
	if scanResult.Status != adapter.StatusSuccess {
		log.Fatalf("Failed to scan: %s", scanResult.Error)
	}

	// Find the contact
	var targetConv *protocol.ConversationRef
	for i := range conversations {
		if conversations[i].DisplayName == contactName {
			targetConv = &conversations[i]
			break
		}
	}

	if targetConv == nil {
		log.Fatalf("Contact '%s' not found in conversation list", contactName)
	}

	fmt.Printf("Sending message to %s: %s\n", contactName, message)

	// First focus on the contact
	focusResult := wechatAdapter.Focus(*targetConv)
	if focusResult.Status != adapter.StatusSuccess {
		log.Fatalf("Failed to focus on contact: %s", focusResult.Error)
	}

	// Wait a bit for UI to stabilize
	time.Sleep(200 * time.Millisecond)

	// Send the message
	sendResult := wechatAdapter.Send(*targetConv, message, "debug-script")

	if *jsonOutput {
		data := map[string]interface{}{
			"contact":        contactName,
			"message":        message,
			"send_success":   sendResult.Status == adapter.StatusSuccess,
			"confidence":     sendResult.Confidence,
			"send_diagnostics": sendResult.Diagnostics,
		}
		jsonData, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(jsonData))
	} else {
		fmt.Printf("Send result:\n")
		fmt.Printf("  Status: %s\n", sendResult.Status)
		fmt.Printf("  Confidence: %.2f\n", sendResult.Confidence)
		fmt.Printf("  Diagnostics:\n")
		for _, diag := range sendResult.Diagnostics {
			fmt.Printf("    %s: %s\n", diag.Level, diag.Message)
			for k, v := range diag.Context {
				fmt.Printf("      %s: %s\n", k, v)
			}
		}
	}
}

func fullTestFlow(wechatAdapter *wechat.WeChatAdapter) {
	fmt.Println("=== WeChat Debugging Full Test Flow ===")
	fmt.Println()

	// Step 1: Find WeChat window
	fmt.Println("Step 1: Finding WeChat window...")
	instances, result := wechatAdapter.Detect()
	if result.Status != adapter.StatusSuccess || len(instances) == 0 {
		log.Fatal("No WeChat window found")
	}
	fmt.Printf("  Found WeChat instance: %s\n", instances[0].InstanceID)
	fmt.Println()

	// Step 2: Scan conversations
	fmt.Println("Step 2: Scanning conversation list...")
	conversations, scanResult := wechatAdapter.Scan(instances[0])
	if scanResult.Status != adapter.StatusSuccess {
		log.Fatalf("Failed to scan: %s", scanResult.Error)
	}
	fmt.Printf("  Found %d conversations\n", len(conversations))
	for i, conv := range conversations {
		fmt.Printf("    [%d] %s\n", i+1, conv.DisplayName)
	}
	fmt.Println()

	// Step 3: Focus on first contact (or specified contact)
	var targetConv *protocol.ConversationRef
	if *contactName != "" {
		for i := range conversations {
			if conversations[i].DisplayName == *contactName {
				targetConv = &conversations[i]
				break
			}
		}
		if targetConv == nil {
			log.Fatalf("Contact '%s' not found", *contactName)
		}
	} else if len(conversations) > 0 {
		targetConv = &conversations[0]
	} else {
		log.Fatal("No conversations found to test")
	}

	fmt.Printf("Step 3: Focusing on contact: %s\n", targetConv.DisplayName)
	focusResult := wechatAdapter.Focus(*targetConv)
	fmt.Printf("  Focus confidence: %.2f\n", focusResult.Confidence)
	fmt.Printf("  Focus diagnostics:\n")
	for _, diag := range focusResult.Diagnostics {
		for k, v := range diag.Context {
			fmt.Printf("    %s: %s\n", k, v)
		}
	}
	fmt.Println()

	// Step 4: Send test message
	fmt.Println("Step 4: Sending test message...")
	time.Sleep(200 * time.Millisecond)
	sendResult := wechatAdapter.Send(*targetConv, *messageContent, "debug-full-test")
	fmt.Printf("  Send confidence: %.2f\n", sendResult.Confidence)
	fmt.Printf("  Send diagnostics:\n")
	for _, diag := range sendResult.Diagnostics {
		for k, v := range diag.Context {
			fmt.Printf("    %s: %s\n", k, v)
		}
	}
	fmt.Println()

	// Step 5: Verify message
	fmt.Println("Step 5: Verifying message delivery...")
	verifyResult, verifyAdapterResult := wechatAdapter.Verify(*targetConv, *messageContent, 5*time.Second)
	if verifyAdapterResult.Status == adapter.StatusSuccess {
		fmt.Printf("  Verify confidence: %.2f\n", verifyAdapterResult.Confidence)
		fmt.Printf("  Verify diagnostics:\n")
		for _, diag := range verifyAdapterResult.Diagnostics {
			for k, v := range diag.Context {
				fmt.Printf("    %s: %s\n", k, v)
			}
		}
	} else {
		fmt.Printf("  Verify failed: %s\n", verifyAdapterResult.Error)
	}
	_ = verifyResult // Keep for future use

	fmt.Println()
	fmt.Println("=== Test Flow Complete ===")
}

func verifyMessage(wechatAdapter *wechat.WeChatAdapter, contactName string, message string) {
	instances, result := wechatAdapter.Detect()
	if result.Status != adapter.StatusSuccess || len(instances) == 0 {
		log.Fatal("No WeChat window found")
	}

	conversations, scanResult := wechatAdapter.Scan(instances[0])
	if scanResult.Status != adapter.StatusSuccess {
		log.Fatalf("Failed to scan: %s", scanResult.Error)
	}

	// Find the contact
	var targetConv *protocol.ConversationRef
	for i := range conversations {
		if conversations[i].DisplayName == contactName {
			targetConv = &conversations[i]
			break
		}
	}

	if targetConv == nil {
		log.Fatalf("Contact '%s' not found in conversation list", contactName)
	}

	fmt.Printf("Verifying message delivery for %s: %s\n", contactName, message)

	// Verify the message
	verifyResult, verifyAdapterResult := wechatAdapter.Verify(*targetConv, message, 5*time.Second)

	if *jsonOutput {
		data := map[string]interface{}{
			"contact":     contactName,
			"message":     message,
			"success":     verifyAdapterResult.Status == adapter.StatusSuccess,
			"confidence":  verifyAdapterResult.Confidence,
			"diagnostics": verifyAdapterResult.Diagnostics,
		}
		jsonData, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(jsonData))
	} else {
		fmt.Printf("Verify result:\n")
		fmt.Printf("  Status: %s\n", verifyAdapterResult.Status)
		fmt.Printf("  Confidence: %.2f\n", verifyAdapterResult.Confidence)
		fmt.Printf("  Diagnostics:\n")
		for _, diag := range verifyAdapterResult.Diagnostics {
			fmt.Printf("    %s: %s\n", diag.Level, diag.Message)
			for k, v := range diag.Context {
				fmt.Printf("      %s: %s\n", k, v)
			}
		}
	}
	_ = verifyResult // Keep for future use
}

func runChain(wechatAdapter *wechat.WeChatAdapter) {
	fmt.Println("=== Running Complete Chain: Scan -> Focus -> Send -> Verify ===")
	fmt.Println()

	// Step 1: Detect WeChat instance
	fmt.Println("Step 1: Detecting WeChat instance...")
	instances, detectResult := wechatAdapter.Detect()
	if detectResult.Status != adapter.StatusSuccess || len(instances) == 0 {
		log.Fatal("No WeChat window found")
	}
	fmt.Printf("  Found WeChat instance: %s\n", instances[0].InstanceID)
	fmt.Println()

	// Step 2: Scan conversations
	fmt.Println("Step 2: Scanning conversation list...")
	conversations, scanResult := wechatAdapter.Scan(instances[0])
	if scanResult.Status != adapter.StatusSuccess {
		log.Fatalf("Failed to scan: %s", scanResult.Error)
	}
	fmt.Printf("  Found %d conversations\n", len(conversations))
	for i, conv := range conversations {
		fmt.Printf("    [%d] %s (position: %d)\n", i+1, conv.DisplayName, conv.ListPosition)
	}
	fmt.Printf("\n  Scan Diagnostics:\n")
	for _, diag := range scanResult.Diagnostics {
		for k, v := range diag.Context {
			fmt.Printf("    %s: %s\n", k, v)
		}
	}
	fmt.Println()

	// Step 3: Find and focus on target contact
	var targetConv *protocol.ConversationRef
	if *contactName != "" {
		for i := range conversations {
			if conversations[i].DisplayName == *contactName {
				targetConv = &conversations[i]
				break
			}
		}
		if targetConv == nil {
			log.Fatalf("Contact '%s' not found", *contactName)
		}
	} else if len(conversations) > 0 {
		targetConv = &conversations[0]
	} else {
		log.Fatal("No conversations found to test")
	}

	fmt.Printf("Step 3: Focusing on contact: %s\n", targetConv.DisplayName)
	focusResult := wechatAdapter.Focus(*targetConv)
	fmt.Printf("  Focus confidence: %.2f\n", focusResult.Confidence)
	fmt.Printf("  Focus Diagnostics:\n")
	for _, diag := range focusResult.Diagnostics {
		for k, v := range diag.Context {
			fmt.Printf("    %s: %s\n", k, v)
		}
	}
	fmt.Println()

	// Step 4: Send message
	fmt.Println("Step 4: Sending message...")
	time.Sleep(200 * time.Millisecond)
	sendResult := wechatAdapter.Send(*targetConv, *messageContent, "debug-chain")
	fmt.Printf("  Send confidence: %.2f\n", sendResult.Confidence)
	fmt.Printf("  Send Diagnostics:\n")
	for _, diag := range sendResult.Diagnostics {
		for k, v := range diag.Context {
			fmt.Printf("    %s: %s\n", k, v)
		}
	}
	fmt.Println()

	// Step 5: Verify message delivery
	fmt.Println("Step 5: Verifying message delivery...")
	verifyResult, verifyAdapterResult := wechatAdapter.Verify(*targetConv, *messageContent, 5*time.Second)
	if verifyAdapterResult.Status == adapter.StatusSuccess {
		fmt.Printf("  Verify confidence: %.2f\n", verifyAdapterResult.Confidence)
		fmt.Printf("  Verify Diagnostics:\n")
		for _, diag := range verifyAdapterResult.Diagnostics {
			for k, v := range diag.Context {
				fmt.Printf("    %s: %s\n", k, v)
			}
		}
	} else {
		fmt.Printf("  Verify failed: %s\n", verifyAdapterResult.Error)
	}
	_ = verifyResult // Keep for future use

	fmt.Println()
	fmt.Println("=== Chain Complete ===")
}
