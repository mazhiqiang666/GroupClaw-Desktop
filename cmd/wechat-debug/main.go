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
	jsonOutput    = flag.Bool("json", false, "Output as JSON")
	contactName   = flag.String("contact", "", "Contact name to focus/send to")
	messageContent = flag.String("message", "Test message from debugging script", "Message content to send")
	mockMode      = flag.Bool("mock", false, "Use mock bridge for testing")
)

// UnifiedExecutor provides common operations for WeChat debugging
type UnifiedExecutor struct {
	adapter *wechat.WeChatAdapter
}

// NewUnifiedExecutor creates a new executor with optional mock mode
func NewUnifiedExecutor(useMock bool) *UnifiedExecutor {
	var adapter *wechat.WeChatAdapter
	if useMock {
		// Create mock bridge for testing
		mockBridge := wechat.NewStateChangingMockBridge()
		mockBridge.SetFindResult([]uintptr{12345})
		adapter = wechat.NewWeChatAdapterWithBridge(mockBridge)
	} else {
		adapter = wechat.NewWeChatAdapter()
	}

	return &UnifiedExecutor{
		adapter: adapter,
	}
}

// Init initializes the adapter
func (e *UnifiedExecutor) Init() adapter.Result {
	result := e.adapter.Init(adapter.Config{})
	if result.Status != adapter.StatusSuccess {
		return result
	}
	return adapter.Result{
		Status:     adapter.StatusSuccess,
		ReasonCode: adapter.ReasonOK,
	}
}

// Destroy cleans up resources
func (e *UnifiedExecutor) Destroy() {
	e.adapter.Destroy()
}

// RunDetect detects WeChat instances
func (e *UnifiedExecutor) RunDetect() ([]protocol.AppInstanceRef, adapter.Result, error) {
	instances, result := e.adapter.Detect()
	if result.Status != adapter.StatusSuccess {
		return nil, result, fmt.Errorf("failed to detect WeChat: %s", result.Error)
	}
	return instances, result, nil
}

// RunScan scans conversations for a given instance
func (e *UnifiedExecutor) RunScan(instance protocol.AppInstanceRef) ([]protocol.ConversationRef, adapter.Result, error) {
	conversations, result := e.adapter.Scan(instance)
	if result.Status != adapter.StatusSuccess {
		return nil, result, fmt.Errorf("failed to scan: %s", result.Error)
	}
	return conversations, result, nil
}

// SelectConversation finds a conversation by name
func (e *UnifiedExecutor) SelectConversation(conversations []protocol.ConversationRef, name string) (*protocol.ConversationRef, error) {
	for i := range conversations {
		if conversations[i].DisplayName == name {
			return &conversations[i], nil
		}
	}
	return nil, fmt.Errorf("contact '%s' not found in conversation list", name)
}

// RunFocus focuses on a conversation
func (e *UnifiedExecutor) RunFocus(conv protocol.ConversationRef) adapter.Result {
	return e.adapter.Focus(conv)
}

// RunSend sends a message to a conversation
func (e *UnifiedExecutor) RunSend(conv protocol.ConversationRef, content string, taskID string) adapter.Result {
	return e.adapter.Send(conv, content, taskID)
}

// RunVerify verifies message delivery
func (e *UnifiedExecutor) RunVerify(conv protocol.ConversationRef, content string, timeout time.Duration) (*protocol.MessageObs, adapter.Result) {
	return e.adapter.Verify(conv, content, timeout)
}

// PrintDiagnostics prints diagnostics in a standardized format
func (e *UnifiedExecutor) PrintDiagnostics(step string, result adapter.Result) {
	fmt.Printf("%s Diagnostics:\n", step)
	for _, diag := range result.Diagnostics {
		for k, v := range diag.Context {
			fmt.Printf("  %s: %s\n", k, v)
		}
	}
}

// StepTrace represents a single step in the execution trace
type StepTrace struct {
	Step         string                  `json:"step"`
	Status       string                  `json:"status"`
	Confidence   float64                 `json:"confidence"`
	Diagnostics  []adapter.Diagnostic    `json:"diagnostics,omitempty"`
	Instances    []protocol.AppInstanceRef `json:"instances,omitempty"`
	Conversations []protocol.ConversationRef `json:"conversations,omitempty"`
	Conversation *protocol.ConversationRef  `json:"conversation,omitempty"`
	Message      *protocol.MessageObs    `json:"message,omitempty"`
	Error        string                  `json:"error,omitempty"`
}

// JSONOutput represents standardized JSON output structure
type JSONOutput struct {
	Mode  string       `json:"mode"`  // "single" or "chain"
	Steps []StepTrace  `json:"steps"`
	Final *StepTrace   `json:"final,omitempty"`
}

// PrintJSON prints standardized JSON output
func (e *UnifiedExecutor) PrintJSON(output JSONOutput) {
	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal JSON: %v", err)
		return
	}
	fmt.Println(string(jsonData))
}

func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
		printUsage()
		return
	}

	command := args[0]

	// Create executor with mock mode if specified
	executor := NewUnifiedExecutor(*mockMode)

	// Initialize adapter
	result := executor.Init()
	if result.Status != adapter.StatusSuccess {
		log.Fatalf("Failed to initialize adapter: %s", result.Error)
	}
	defer executor.Destroy()

	switch command {
	case "find-window":
		findWeChatWindow(executor)
	case "list-nodes":
		if len(args) < 2 {
			log.Fatal("Usage: wechat-debug list-nodes <window-handle>")
		}
		handle, err := strconv.ParseUint(args[1], 10, 64)
		if err != nil {
			log.Fatalf("Invalid window handle: %v", err)
		}
		listNodes(executor, uintptr(handle))
	case "scan":
		scanConversations(executor)
	case "focus":
		if *contactName == "" {
			log.Fatal("Usage: wechat-debug focus --contact <name>")
		}
		focusContact(executor, *contactName)
	case "send":
		if *contactName == "" {
			log.Fatal("Usage: wechat-debug send --contact <name> [--message <content>]")
		}
		sendMessage(executor, *contactName, *messageContent)
	case "verify":
		if *contactName == "" {
			log.Fatal("Usage: wechat-debug verify --contact <name> [--message <content>]")
		}
		verifyMessage(executor, *contactName, *messageContent)
	case "full-test":
		fullTestFlow(executor)
	case "run-chain":
		runChain(executor)
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
	fmt.Println("  --mock                                - Use mock bridge for testing")
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
	fmt.Println("  wechat-debug full-test --mock")
	fmt.Println("  wechat-debug run-chain --contact \"张三\" --message \"Test message\" --mock")
}

func findWeChatWindow(executor *UnifiedExecutor) {
	instances, result, err := executor.RunDetect()
	if err != nil {
		log.Fatalf("Failed to detect WeChat: %s", err)
	}

	if *jsonOutput {
		step := StepTrace{
			Step:       "find-window",
			Status:     string(result.Status),
			Confidence: result.Confidence,
			Instances:  instances,
		}
		output := JSONOutput{
			Mode:  "single",
			Steps: []StepTrace{step},
			Final: &step,
		}
		executor.PrintJSON(output)
	} else {
		fmt.Printf("Found %d WeChat instance(s):\n", len(instances))
		for i, inst := range instances {
			fmt.Printf("  [%d] AppID: %s, InstanceID: %s\n", i+1, inst.AppID, inst.InstanceID)
		}
	}
}

func listNodes(executor *UnifiedExecutor, windowHandle uintptr) {
	instances, _, err := executor.RunDetect()
	if err != nil || len(instances) == 0 {
		log.Fatal("No WeChat window found")
	}

	conversations, scanResult, err := executor.RunScan(instances[0])
	if err != nil {
		log.Fatalf("Failed to scan: %s", err)
	}

	if *jsonOutput {
		step := StepTrace{
			Step:          "list-nodes",
			Status:        string(scanResult.Status),
			Confidence:    scanResult.Confidence,
			Conversations: conversations,
			Diagnostics:   scanResult.Diagnostics,
		}
		output := JSONOutput{
			Mode:  "single",
			Steps: []StepTrace{step},
			Final: &step,
		}
		executor.PrintJSON(output)
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

func scanConversations(executor *UnifiedExecutor) {
	instances, _, err := executor.RunDetect()
	if err != nil || len(instances) == 0 {
		log.Fatal("No WeChat window found")
	}

	conversations, scanResult, err := executor.RunScan(instances[0])
	if err != nil {
		log.Fatalf("Failed to scan: %s", err)
	}

	if *jsonOutput {
		step := StepTrace{
			Step:          "scan",
			Status:        string(scanResult.Status),
			Confidence:    scanResult.Confidence,
			Conversations: conversations,
			Diagnostics:   scanResult.Diagnostics,
		}
		output := JSONOutput{
			Mode:  "single",
			Steps: []StepTrace{step},
			Final: &step,
		}
		executor.PrintJSON(output)
	} else {
		fmt.Printf("Scan completed:\n")
		fmt.Printf("  Conversations found: %d\n", len(conversations))
		for i, conv := range conversations {
			fmt.Printf("  [%d] %s (position: %d)\n", i+1, conv.DisplayName, conv.ListPosition)
		}
		fmt.Printf("\nDiagnostics:\n")
		executor.PrintDiagnostics("Scan", scanResult)
	}
}

func focusContact(executor *UnifiedExecutor, contactName string) {
	instances, _, err := executor.RunDetect()
	if err != nil || len(instances) == 0 {
		log.Fatal("No WeChat window found")
	}

	conversations, _, err := executor.RunScan(instances[0])
	if err != nil {
		log.Fatalf("Failed to scan: %s", err)
	}

	targetConv, err := executor.SelectConversation(conversations, contactName)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Focusing on contact: %s\n", contactName)

	focusResult := executor.RunFocus(*targetConv)

	if *jsonOutput {
		step := StepTrace{
			Step:         "focus",
			Status:       string(focusResult.Status),
			Confidence:   focusResult.Confidence,
			Conversation: targetConv,
			Diagnostics:  focusResult.Diagnostics,
		}
		output := JSONOutput{
			Mode:  "single",
			Steps: []StepTrace{step},
			Final: &step,
		}
		executor.PrintJSON(output)
	} else {
		fmt.Printf("Focus result:\n")
		fmt.Printf("  Status: %s\n", focusResult.Status)
		fmt.Printf("  Confidence: %.2f\n", focusResult.Confidence)
		executor.PrintDiagnostics("Focus", focusResult)
	}
}

func sendMessage(executor *UnifiedExecutor, contactName string, message string) {
	instances, _, err := executor.RunDetect()
	if err != nil || len(instances) == 0 {
		log.Fatal("No WeChat window found")
	}

	conversations, _, err := executor.RunScan(instances[0])
	if err != nil {
		log.Fatalf("Failed to scan: %s", err)
	}

	targetConv, err := executor.SelectConversation(conversations, contactName)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Sending message to %s: %s\n", contactName, message)

	// First focus on the contact
	focusResult := executor.RunFocus(*targetConv)
	if focusResult.Status != adapter.StatusSuccess {
		log.Fatalf("Failed to focus on contact: %s", focusResult.Error)
	}

	// Wait a bit for UI to stabilize
	time.Sleep(200 * time.Millisecond)

	// Send the message
	sendResult := executor.RunSend(*targetConv, message, "debug-script")

	if *jsonOutput {
		step := StepTrace{
			Step:         "send",
			Status:       string(sendResult.Status),
			Confidence:   sendResult.Confidence,
			Conversation: targetConv,
			Diagnostics:  sendResult.Diagnostics,
		}
		output := JSONOutput{
			Mode:  "single",
			Steps: []StepTrace{step},
			Final: &step,
		}
		executor.PrintJSON(output)
	} else {
		fmt.Printf("Send result:\n")
		fmt.Printf("  Status: %s\n", sendResult.Status)
		fmt.Printf("  Confidence: %.2f\n", sendResult.Confidence)
		executor.PrintDiagnostics("Send", sendResult)
	}
}

func verifyMessage(executor *UnifiedExecutor, contactName string, message string) {
	instances, _, err := executor.RunDetect()
	if err != nil || len(instances) == 0 {
		log.Fatal("No WeChat window found")
	}

	conversations, _, err := executor.RunScan(instances[0])
	if err != nil {
		log.Fatalf("Failed to scan: %s", err)
	}

	targetConv, err := executor.SelectConversation(conversations, contactName)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Verifying message delivery for %s: %s\n", contactName, message)

	// Verify the message
	verifyResult, verifyAdapterResult := executor.RunVerify(*targetConv, message, 5*time.Second)

	if *jsonOutput {
		step := StepTrace{
			Step:         "verify",
			Status:       string(verifyAdapterResult.Status),
			Confidence:   verifyAdapterResult.Confidence,
			Conversation: targetConv,
			Message:      verifyResult,
			Diagnostics:  verifyAdapterResult.Diagnostics,
		}
		output := JSONOutput{
			Mode:  "single",
			Steps: []StepTrace{step},
			Final: &step,
		}
		executor.PrintJSON(output)
	} else {
		fmt.Printf("Verify result:\n")
		fmt.Printf("  Status: %s\n", verifyAdapterResult.Status)
		fmt.Printf("  Confidence: %.2f\n", verifyAdapterResult.Confidence)
		executor.PrintDiagnostics("Verify", verifyAdapterResult)
	}
}

func fullTestFlow(executor *UnifiedExecutor) {
	fmt.Println("=== WeChat Debugging Full Test Flow ===")
	fmt.Println()

	// Track all steps for JSON output
	var steps []StepTrace

	// Step 1: Find WeChat window
	fmt.Println("Step 1: Finding WeChat window...")
	instances, detectResult, err := executor.RunDetect()
	if err != nil || len(instances) == 0 {
		log.Fatal("No WeChat window found")
	}
	fmt.Printf("  Found WeChat instance: %s\n", instances[0].InstanceID)
	fmt.Println()

	step1 := StepTrace{
		Step:       "detect",
		Status:     string(detectResult.Status),
		Confidence: detectResult.Confidence,
		Instances:  instances,
		Diagnostics: detectResult.Diagnostics,
	}
	steps = append(steps, step1)

	// Step 2: Scan conversations
	fmt.Println("Step 2: Scanning conversation list...")
	conversations, scanResult, err := executor.RunScan(instances[0])
	if err != nil {
		log.Fatalf("Failed to scan: %s", err)
	}
	fmt.Printf("  Found %d conversations\n", len(conversations))
	for i, conv := range conversations {
		fmt.Printf("    [%d] %s\n", i+1, conv.DisplayName)
	}
	fmt.Println()

	step2 := StepTrace{
		Step:          "scan",
		Status:        string(scanResult.Status),
		Confidence:    scanResult.Confidence,
		Conversations: conversations,
		Diagnostics:   scanResult.Diagnostics,
	}
	steps = append(steps, step2)

	// Step 3: Focus on first contact (or specified contact)
	var targetConv *protocol.ConversationRef
	if *contactName != "" {
		targetConv, err = executor.SelectConversation(conversations, *contactName)
		if err != nil {
			log.Fatal(err)
		}
	} else if len(conversations) > 0 {
		targetConv = &conversations[0]
	} else {
		log.Fatal("No conversations found to test")
	}

	fmt.Printf("Step 3: Focusing on contact: %s\n", targetConv.DisplayName)
	focusResult := executor.RunFocus(*targetConv)
	fmt.Printf("  Focus confidence: %.2f\n", focusResult.Confidence)
	executor.PrintDiagnostics("Focus", focusResult)
	fmt.Println()

	step3 := StepTrace{
		Step:         "focus",
		Status:       string(focusResult.Status),
		Confidence:   focusResult.Confidence,
		Conversation: targetConv,
		Diagnostics:  focusResult.Diagnostics,
	}
	steps = append(steps, step3)

	// Step 4: Send test message
	fmt.Println("Step 4: Sending test message...")
	time.Sleep(200 * time.Millisecond)
	sendResult := executor.RunSend(*targetConv, *messageContent, "debug-full-test")
	fmt.Printf("  Send confidence: %.2f\n", sendResult.Confidence)
	executor.PrintDiagnostics("Send", sendResult)
	fmt.Println()

	step4 := StepTrace{
		Step:         "send",
		Status:       string(sendResult.Status),
		Confidence:   sendResult.Confidence,
		Conversation: targetConv,
		Diagnostics:  sendResult.Diagnostics,
	}
	steps = append(steps, step4)

	// Step 5: Verify message
	fmt.Println("Step 5: Verifying message delivery...")
	verifyResult, verifyAdapterResult := executor.RunVerify(*targetConv, *messageContent, 5*time.Second)
	if verifyAdapterResult.Status == adapter.StatusSuccess {
		fmt.Printf("  Verify confidence: %.2f\n", verifyAdapterResult.Confidence)
		executor.PrintDiagnostics("Verify", verifyAdapterResult)
	} else {
		fmt.Printf("  Verify failed: %s\n", verifyAdapterResult.Error)
	}
	_ = verifyResult // Keep for future use

	fmt.Println()
	fmt.Println("=== Test Flow Complete ===")

	// Print standardized JSON output if requested
	if *jsonOutput {
		step5 := StepTrace{
			Step:         "verify",
			Status:       string(verifyAdapterResult.Status),
			Confidence:   verifyAdapterResult.Confidence,
			Conversation: targetConv,
			Message:      verifyResult,
			Diagnostics:  verifyAdapterResult.Diagnostics,
		}
		steps = append(steps, step5)

		output := JSONOutput{
			Mode:  "chain",
			Steps: steps,
			Final: &step5,
		}
		executor.PrintJSON(output)
	}
}

func runChain(executor *UnifiedExecutor) {
	fmt.Println("=== Running Complete Chain: Scan -> Focus -> Send -> Verify ===")
	fmt.Println()

	// Track all steps for JSON output
	var steps []StepTrace

	// Step 1: Detect WeChat instance
	fmt.Println("Step 1: Detecting WeChat instance...")
	instances, detectResult, err := executor.RunDetect()
	if err != nil || len(instances) == 0 {
		log.Fatal("No WeChat window found")
	}
	fmt.Printf("  Found WeChat instance: %s\n", instances[0].InstanceID)
	fmt.Println()

	step1 := StepTrace{
		Step:       "detect",
		Status:     string(detectResult.Status),
		Confidence: detectResult.Confidence,
		Instances:  instances,
		Diagnostics: detectResult.Diagnostics,
	}
	steps = append(steps, step1)

	// Step 2: Scan conversations
	fmt.Println("Step 2: Scanning conversation list...")
	conversations, scanResult, err := executor.RunScan(instances[0])
	if err != nil {
		log.Fatalf("Failed to scan: %s", err)
	}
	fmt.Printf("  Found %d conversations\n", len(conversations))
	for i, conv := range conversations {
		fmt.Printf("    [%d] %s (position: %d)\n", i+1, conv.DisplayName, conv.ListPosition)
	}
	fmt.Printf("\n  Scan Diagnostics:\n")
	executor.PrintDiagnostics("Scan", scanResult)
	fmt.Println()

	step2 := StepTrace{
		Step:          "scan",
		Status:        string(scanResult.Status),
		Confidence:    scanResult.Confidence,
		Conversations: conversations,
		Diagnostics:   scanResult.Diagnostics,
	}
	steps = append(steps, step2)

	// Step 3: Find and focus on target contact
	var targetConv *protocol.ConversationRef
	if *contactName != "" {
		targetConv, err = executor.SelectConversation(conversations, *contactName)
		if err != nil {
			log.Fatal(err)
		}
	} else if len(conversations) > 0 {
		targetConv = &conversations[0]
	} else {
		log.Fatal("No conversations found to test")
	}

	fmt.Printf("Step 3: Focusing on contact: %s\n", targetConv.DisplayName)
	focusResult := executor.RunFocus(*targetConv)
	fmt.Printf("  Focus confidence: %.2f\n", focusResult.Confidence)
	fmt.Printf("  Focus Diagnostics:\n")
	executor.PrintDiagnostics("Focus", focusResult)
	fmt.Println()

	step3 := StepTrace{
		Step:         "focus",
		Status:       string(focusResult.Status),
		Confidence:   focusResult.Confidence,
		Conversation: targetConv,
		Diagnostics:  focusResult.Diagnostics,
	}
	steps = append(steps, step3)

	// Step 4: Send message
	fmt.Println("Step 4: Sending message...")
	time.Sleep(200 * time.Millisecond)
	sendResult := executor.RunSend(*targetConv, *messageContent, "debug-chain")
	fmt.Printf("  Send confidence: %.2f\n", sendResult.Confidence)
	fmt.Printf("  Send Diagnostics:\n")
	executor.PrintDiagnostics("Send", sendResult)
	fmt.Println()

	step4 := StepTrace{
		Step:         "send",
		Status:       string(sendResult.Status),
		Confidence:   sendResult.Confidence,
		Conversation: targetConv,
		Diagnostics:  sendResult.Diagnostics,
	}
	steps = append(steps, step4)

	// Step 5: Verify message delivery
	fmt.Println("Step 5: Verifying message delivery...")
	verifyResult, verifyAdapterResult := executor.RunVerify(*targetConv, *messageContent, 5*time.Second)
	if verifyAdapterResult.Status == adapter.StatusSuccess {
		fmt.Printf("  Verify confidence: %.2f\n", verifyAdapterResult.Confidence)
		fmt.Printf("  Verify Diagnostics:\n")
		executor.PrintDiagnostics("Verify", verifyAdapterResult)
	} else {
		fmt.Printf("  Verify failed: %s\n", verifyAdapterResult.Error)
	}
	_ = verifyResult // Keep for future use

	fmt.Println()
	fmt.Println("=== Chain Complete ===")

	// Print standardized JSON output if requested
	if *jsonOutput {
		step5 := StepTrace{
			Step:         "verify",
			Status:       string(verifyAdapterResult.Status),
			Confidence:   verifyAdapterResult.Confidence,
			Conversation: targetConv,
			Message:      verifyResult,
			Diagnostics:  verifyAdapterResult.Diagnostics,
		}
		steps = append(steps, step5)

		output := JSONOutput{
			Mode:  "chain",
			Steps: steps,
			Final: &step5,
		}
		executor.PrintJSON(output)
	}
}
