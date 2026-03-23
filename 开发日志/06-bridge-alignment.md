# Bridge Alignment Documentation

This document defines the alignment between mock bridges and real Windows bridge for WeChat adapter testing.

## 1. Mock Bridge Implementations

### 1.1 Exported Mock Bridges (mock_bridge.go)

The `internal/agent/adapter/wechat/mock_bridge.go` file exports two mock bridge implementations for external use:

| Mock Bridge | Purpose | Location |
|-------------|---------|----------|
| `ControlledMockBridge` | Minimal closed-loop testing with full control | `mock_bridge.go` |
| `StateChangingMockBridge` | Stateful testing with realistic node changes | `mock_bridge.go` |

### 1.2 Internal Mock Bridges (Test Files)

Test files also contain internal mock bridge implementations:

| Mock Bridge | Purpose | Location |
|-------------|---------|----------|
| `controlledMockBridge` | Internal test mock (lowercase) | `adapter_diagnostic_test.go` |
| `stateChangingMockBridge` | Internal test mock (lowercase) | `diagnostic_flow_test.go` |

**Note:** The internal mocks in test files are lowercase and not exported. Use the exported versions from `mock_bridge.go` for external debugging tools.

## 2. AccessibleNode Structure Comparison

### 1.1 Real Bridge Structure (internal/agent/windows/interface.go)

```go
type AccessibleNode struct {
    Handle      uintptr
    Name        string
    Role        string
    Value       string
    ClassName   string
    Bounds      [4]int // x, y, width, height
    Children    []AccessibleNode
    TreePath    string // Hierarchical path like [0].[3].[2]
}
```

### 1.2 Real Bridge Field Population (bridge.go)

| Field | Populated? | Source | Notes |
|-------|------------|--------|-------|
| Handle | Yes | Implicit | Node identifier |
| Name | Yes | get_accName() | Node display name |
| Role | Yes | get_accRole() | Node role string |
| Value | No | - | Not retrieved by real bridge |
| ClassName | Empty string | - | IAccessible has no get_accClassName |
| Bounds | Yes | accLocation() | [x, y, width, height] |
| Children | Yes | Recursive enumeration | Populated in enumerateAccessibleChildren |
| TreePath | No | - | Not set by real bridge |

### 1.3 Mock Bridge Comparison (Exported vs Internal)

| Field | StateChangingMockBridge (exported) | stateChangingMockBridge (internal) | Real Bridge |
|-------|-------------------------------------|------------------------------------|-------------|
| Handle | ✅ | ✅ | ✅ |
| Name | ✅ | ✅ | ✅ |
| Role | ✅ | ✅ | ✅ |
| Value | ❌ | ❌ | ❌ (not populated) |
| ClassName | ✅ | ✅ | ✅ (empty string) |
| Bounds | ✅ | ✅ | ✅ |
| Children | ✅ (empty slice) | ❌ | ✅ |
| TreePath | ✅ | ✅ | ❌ (not populated) |

### 1.4 AccessibleNode Field Population

| Field | Real Bridge | Exported Mock | Internal Mock | Notes |
|-------|-------------|---------------|---------------|-------|
| Handle | ✅ Implicit | ✅ | ✅ | Node identifier |
| Name | ✅ get_accName() | ✅ | ✅ | Node display name |
| Role | ✅ get_accRole() | ✅ | ✅ | Node role string |
| Value | ❌ Not retrieved | ❌ | ❌ | Not used by adapter |
| ClassName | ✅ Empty string | ✅ | ✅ | IAccessible has no get_accClassName |
| Bounds | ✅ accLocation() | ✅ | ✅ | [x, y, width, height] |
| Children | ✅ Recursive enum | ✅ Empty slice | ❌ | Populated in enumerateAccessibleChildren |
| TreePath | ❌ Not set | ✅ | ✅ | Optional, for debugging |

### 1.5 Issues Identified and Fixed

**Exported Mock Bridges (mock_bridge.go):**
- ✅ All fields properly populated
- ✅ ClassName set to empty string (matching real bridge)
- ✅ Children set to empty slice for leaf nodes
- ⚠️ TreePath populated (optional, real bridge doesn't set it)

**Internal Mock Bridges (test files):**
- `stateChangingMockBridge`: Missing `Children` field in node definitions
- `controlledMockBridge`: Missing `ClassName`, `Children`, `TreePath` fields

## 2. Mock/Real Mode Comparison (wechat-debug)

### 2.1 Mode Switching

The `wechat-debug` tool supports two modes:

| Mode | Flag | Bridge Used | Use Case |
|------|------|-------------|----------|
| Mock | `--mock` | `StateChangingMockBridge` | Testing without real WeChat |
| Real | (default) | Real Windows Bridge | Testing with actual WeChat |

### 2.2 Mock Mode Behavior

When using `--mock` flag:

```go
mockBridge := wechat.NewStateChangingMockBridge()
mockBridge.SetFindResult([]uintptr{12345})
adapter := wechat.NewWeChatAdapterWithBridge(mockBridge)
```

**State Progression:**
1. **Initial State**: Returns `nodesInitial` (2 conversations: 张三, 李四)
2. **After Focus**: Returns `nodesAfterFocus` (3 nodes: 2 conversations + message area)
3. **After Send**: Returns `nodesAfterSend` (4 nodes: 2 conversations + message area + new message)

### 2.3 Real Mode Behavior

When using real mode (default):

```go
adapter := wechat.NewWeChatAdapter()
```

**Behavior:**
- Uses actual Windows Bridge (`internal/agent/windows/bridge.go`)
- Interacts with real WeChat application
- Requires WeChat to be running

### 2.4 Diagnostic Output Comparison

Both modes produce identical diagnostic output format:

```json
{
  "step": "focus",
  "status": "success",
  "confidence": "0.85",
  "diagnostics": [
    {
      "timestamp": "2026-03-23T10:00:00Z",
      "level": "info",
      "message": "Focus completed",
      "context": {
        "locate_source": "tree_path_name",
        "evidence_count": "3",
        "confidence": "0.85"
      }
    }
  ]
}
```

## 3. Bridge Output Format Comparison

### 2.1 Real Bridge Output (bridge-dump)

**Text Format** (printNodesText):
```
[path] Handle: %d, Name: %s, Role: %s, Class: %s Bounds(x=%d,y=%d,w=%d,h=%d)
```

**JSON Format** (printNodesJSON):
```json
{
  "handle": <int>,
  "name": <string>,
  "role": <string>,
  "className": <string>,
  "bounds": {
    "x": <int>,
    "y": <int>,
    "width": <int>,
    "height": <int>
  },
  "children": [...] // if any
}
```

### 2.2 Diagnostic Field Alignment

| Diagnostic Field | Mock Implementation | Real Implementation | Status |
|------------------|---------------------|---------------------|--------|
| locate_source | ✅ | ✅ | Aligned |
| evidence_count | ✅ | ✅ | Aligned |
| confidence | ✅ (2 decimal) | ✅ (2 decimal) | Aligned |
| delivery_state | ✅ | ✅ | Aligned |
| new_message_nodes | ✅ | ✅ | Aligned |
| message_content_match | ✅ | ✅ | Aligned |
| node_still_exists | ✅ | ✅ | Aligned |
| node_has_active_state | ✅ | ✅ | Aligned |

## 3. Required Fixes

### 3.1 stateChangingMockBridge (diagnostic_flow_test.go)

**Issues:**
1. `nodesAfterSend` nodes are missing `ClassName` and `TreePath` fields
2. No `Children` field in any node definitions

**Fix:**
- Add `ClassName` and `TreePath` to `nodesAfterSend` nodes
- Add `Children` field to all node definitions (empty slice for simplicity)

### 3.2 controlledMockBridge (adapter_diagnostic_test.go)

**Issues:**
1. Missing `ClassName`, `Children`, `TreePath` fields

**Fix:**
- Add `ClassName` field (empty string to match real bridge)
- Add `Children` field (empty slice)
- Add `TreePath` field (optional, for consistency)

## 4. Alignment Rules

### 4.1 Node Structure Rules

1. **All mock nodes must include all AccessibleNode fields**
2. **ClassName should be empty string** (matching real bridge behavior)
3. **Children should be empty slice** (for leaf nodes)
4. **TreePath is optional** (real bridge doesn't populate it)

### 4.2 Diagnostic Field Rules

1. **Confidence values**: Always 2 decimal places (e.g., "0.85", "1.00")
2. **Boolean values**: Always lowercase strings ("true", "false")
3. **Integer values**: Always string format (e.g., "3", "1")
4. **Messages**: Semicolon-separated list for multiple messages

### 4.3 Output Format Rules

1. **Text format**: `Handle: %d, Name: %s, Role: %s, Class: %s Bounds(x=%d,y=%d,w=%d,h=%d)`
2. **JSON format**: Include handle, name, role, className, bounds, children
3. **Children**: Only include if non-empty

## 5. Alignment Test Examples

### 5.1 Diagnostic Flow Tests

The `diagnostic_flow_test.go` file contains comprehensive alignment tests:

```go
// TestDiagnosticFlow_CompleteChain tests the complete chain:
// Scan -> Focus -> Send -> Verify with state-changing mock
func TestDiagnosticFlow_CompleteChain(t *testing.T) {
    mock := newStateChangingMockBridge()
    mock.findResult = []uintptr{12345}
    wechatAdapter := NewWeChatAdapterWithBridge(mock)

    // Step 1: Scan conversations
    conversations, scanResult := wechatAdapter.Scan(...)

    // Step 2: Focus on conversation
    focusResult := wechatAdapter.Focus(conv)

    // Step 3: Send message
    sendResult := wechatAdapter.Send(conv, "Test message", "task-123")

    // Step 4: Verify message delivery
    msg, verifyResult := wechatAdapter.Verify(conv, "Test message", 5*time.Second)
}
```

**Verification Points:**
- ✅ All operations return `StatusSuccess`
- ✅ Diagnostic fields present: `locate_source`, `evidence_count`, `confidence`
- ✅ Confidence format: 2 decimal places (e.g., "0.85")
- ✅ State changes tracked correctly

### 5.2 Minimum Closed-Loop Tests

The `adapter_diagnostic_test.go` file contains minimal closed-loop tests:

```go
func TestWeChatAdapter_MinimumClosedLoop_Diagnostics(t *testing.T) {
    mock := newControlledMockBridge()
    mock.findResult = []uintptr{12345}
    wechatAdapter := NewWeChatAdapterWithBridge(mock)

    // Test complete flow
    instances, _ := wechatAdapter.Detect()
    conversations, _ := wechatAdapter.Scan(instances[0])
    conv := conversations[0]

    focusResult := wechatAdapter.Focus(conv)
    sendResult := wechatAdapter.Send(conv, "Test message", "task-123")
    _, verifyResult := wechatAdapter.Verify(conv, "Test message", 5)

    // Verify diagnostics structure
    evidence := FocusVerificationEvidence{
        LocateSource:       "tree_path_name",
        NodeStillExists:    true,
        NodeHasActiveState: true,
        Confidence:         0.85,
        EvidenceCount:      3,
    }

    diagnostics := ConvertFocusEvidenceToDiagnostics(evidence)

    // Verify required fields
    requiredFields := []string{"locate_source", "evidence_count", "confidence"}
    for _, field := range requiredFields {
        if _, ok := diagnostics[field]; !ok {
            t.Errorf("Missing required diagnostic field: %s", field)
        }
    }
}
```

### 5.3 Mock/Real Alignment Test

To verify mock and real bridges produce consistent diagnostics:

```go
func TestMockRealAlignment(t *testing.T) {
    // Test with mock bridge
    mockBridge := wechat.NewStateChangingMockBridge()
    mockAdapter := wechat.NewWeChatAdapterWithBridge(mockBridge)

    // Test with real bridge (if available)
    realAdapter := wechat.NewWeChatAdapter()

    // Compare diagnostic output format
    // Both should produce identical field names and formats
}
```

## 6. Test Verification

### 6.1 Run Alignment Tests

```bash
# Run diagnostic flow tests
go test -v ./internal/agent/adapter/wechat -run "TestDiagnosticFlow" -timeout 30s

# Run minimum closed-loop tests
go test -v ./internal/agent/adapter/wechat -run "TestWeChatAdapter_MinimumClosedLoop" -timeout 30s

# Run all WeChat adapter tests
make test-adapter
```

### 6.2 Verify Mock Bridge Output

```bash
# Compare mock bridge output with expected format
# The mock should produce identical field names and formats as real bridge

# Test with mock mode
wechat-debug run-chain --contact "张三" --message "Test message" --mock --json

# Test with real mode (requires WeChat running)
wechat-debug run-chain --contact "张三" --message "Test message" --json
```

## 7. References

### 7.1 Core Files

- `internal/agent/windows/interface.go`: AccessibleNode structure definition
- `internal/agent/windows/bridge.go`: Real bridge implementation
- `internal/agent/adapter/wechat/mock_bridge.go`: Exported mock bridge implementations

### 7.2 Test Files

- `internal/agent/adapter/wechat/diagnostic_flow_test.go`: State-changing mock bridge tests
- `internal/agent/adapter/wechat/adapter_diagnostic_test.go`: Minimum closed-loop tests

### 7.3 Debug Tools

- `cmd/wechat-debug/main.go`: WeChat debugging tool with mock/real mode support
- `cmd/bridge-dump/main.go`: Real bridge diagnostic output tool

### 7.4 Documentation

- `开发日志/06-bridge-alignment.md`: This document
- `README.md`: Project overview and联调 command examples
