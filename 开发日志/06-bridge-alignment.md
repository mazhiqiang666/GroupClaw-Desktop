# Bridge Alignment Documentation

This document defines the alignment between mock bridges and real Windows bridge for WeChat adapter testing.

## 1. AccessibleNode Structure Comparison

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

### 1.3 Mock Bridge Comparison

| Field | stateChangingMockBridge | controlledMockBridge | Real Bridge |
|-------|-------------------------|----------------------|-------------|
| Handle | ✅ | ✅ | ✅ |
| Name | ✅ | ✅ | ✅ |
| Role | ✅ | ✅ | ✅ |
| Value | ❌ | ❌ | ❌ (not populated) |
| ClassName | ✅ | ❌ | ✅ (empty string) |
| Bounds | ✅ | ✅ | ✅ |
| Children | ❌ | ❌ | ✅ |
| TreePath | ✅ | ❌ | ❌ (not populated) |

### 1.4 Issues Identified

1. **stateChangingMockBridge**:
   - Missing `Children` field in node definitions
   - `nodesAfterSend` is missing `ClassName` and `TreePath` fields
   - Has `TreePath` which real bridge doesn't populate

2. **controlledMockBridge**:
   - Missing `ClassName`, `Children`, `TreePath` fields
   - Only has basic fields: Handle, Name, Role, Bounds

## 2. Bridge Output Format Comparison

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

## 5. Test Verification

### 5.1 Run Alignment Tests

```bash
# Run diagnostic flow tests
go test -v ./internal/agent/adapter/wechat -run "TestDiagnosticFlow" -timeout 30s

# Run minimum closed-loop tests
go test -v ./internal/agent/adapter/wechat -run "TestWeChatAdapter_MinimumClosedLoop" -timeout 30s

# Run all WeChat adapter tests
make test-adapter
```

### 5.2 Verify Mock Bridge Output

```bash
# Compare mock bridge output with expected format
# The mock should produce identical field names and formats as real bridge
```

## 6. References

- `internal/agent/windows/interface.go`: AccessibleNode structure definition
- `internal/agent/windows/bridge.go`: Real bridge implementation
- `cmd/bridge-dump/main.go`: Real bridge diagnostic output
- `internal/agent/adapter/wechat/diagnostic_flow_test.go`: stateChangingMockBridge
- `internal/agent/adapter/wechat/adapter_diagnostic_test.go`: controlledMockBridge
