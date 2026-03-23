# Diagnostics Schema Documentation

This document defines the complete diagnostics schema for WeChat adapter operations (Focus, Send, Verify).

## Overview

The diagnostics system provides detailed verification evidence for each operation, enabling:
- Transparent operation verification
- Confidence scoring
- Debugging and troubleshooting
- Consistent output format across mock and real implementations

## 1. Focus Operation Diagnostics

### 1.1 FocusVerificationEvidence Structure

```go
type FocusVerificationEvidence struct {
    // Node-level evidence
    NodeStillExists      bool
    NodeHasActiveState   bool
    NodeBoundsMatch      bool

    // Title-level evidence
    TitleContainsTarget  bool
    TitleChanged         bool

    // Panel-level evidence
    PanelSwitchDetected  bool
    MessageAreaVisible   bool

    // Confidence and source
    Confidence    float64
    LocateSource  string
    EvidenceCount int
}
```

### 1.2 Diagnostic Context Fields

| Field | Type | Description | Example Values |
|-------|------|-------------|----------------|
| `locate_source` | string | Source used to locate the conversation | `tree_path_name`, `bounds_match`, `stable_key`, `name_match`, `not_found` |
| `node_still_exists` | bool | Whether target node exists in tree | `true`, `false` |
| `node_has_active_state` | bool | Whether node shows active/selected state | `true`, `false` |
| `title_contains_target` | bool | Whether window title contains target name | `true`, `false` |
| `panel_switch_detected` | bool | Whether panel switch was detected | `true`, `false` |
| `message_area_visible` | bool | Whether message area is visible | `true`, `false` |
| `evidence_count` | string | Count of positive evidence items | `"3"`, `"5"` |
| `confidence` | string | Confidence score (2 decimal places) | `"0.85"`, `"1.00"` |

### 1.3 Locate Source Values

| Source | Description | Confidence |
|--------|-------------|------------|
| `tree_path_name` | Matched using tree path + name | 1.0 |
| `bounds_match` | Matched using exact bounds | 1.0 |
| `stable_key` | Matched using stable key (PreviewText) | 1.0 |
| `name_match` | Matched using name only | 0.8 |
| `not_found` | No match found | 0.0 |

### 1.4 Confidence Calculation

Focus confidence is calculated based on weighted evidence:

| Evidence | Weight | Description |
|----------|--------|-------------|
| Node Still Exists | 20% | Target node found in tree |
| Node Has Active State | 30% | Node shows selected/active state |
| Title Contains Target | 20% | Window title includes contact name |
| Panel Switch Detected | 15% | Node count changed significantly |
| Message Area Visible | 15% | Message area node found |

**Formula:**
```
confidence = (sum of positive evidence weights) / (sum of all evidence weights)
```

**Example:**
- Node exists: ✓ (20%)
- Active state: ✓ (30%)
- Title match: ✗ (0%)
- Panel switch: ✗ (0%)
- Message area: ✓ (15%)
- **Confidence = (20 + 30 + 15) / 100 = 0.65**

## 2. Send Operation Diagnostics

### 2.1 SendVerificationEvidence Structure

```go
type SendVerificationEvidence struct {
    // Node-level evidence
    NewMessageNodes      int
    MessageNodeAdded     bool
    MessageContentMatch  bool

    // Screenshot-level evidence
    ScreenshotChanged    bool
    ChatAreaDiff         float64

    // Confidence
    Confidence           float64
}
```

### 2.2 Diagnostic Context Fields

| Field | Type | Description | Example Values |
|-------|------|-------------|----------------|
| `new_message_nodes` | string | Number of new message nodes detected | `"1"`, `"0"` |
| `message_node_added` | bool | Whether a new message node was added | `true`, `false` |
| `message_content_match` | bool | Whether new node contains sent content | `true`, `false` |
| `screenshot_changed` | bool | Whether screenshot changed after send | `true`, `false` |
| `chat_area_diff` | string | Pixel difference in chat area (2 decimal) | `"0.05"`, `"0.00"` |
| `confidence` | string | Confidence score (2 decimal places) | `"0.90"`, `"0.75"` |

### 2.3 Confidence Calculation

Send confidence is calculated based on weighted evidence:

| Evidence | Weight | Description |
|----------|--------|-------------|
| Message Node Added | 40% | New node detected in message area |
| Message Content Match | 30% | New node contains sent message |
| Screenshot Changed | 20% | Visual change detected |
| Chat Area Diff | 10% | Pixel difference > 1% |

**Formula:**
```
confidence = (sum of positive evidence weights) / (sum of all evidence weights)
```

**Example:**
- New node added: ✓ (40%)
- Content match: ✓ (30%)
- Screenshot changed: ✓ (20%)
- Chat area diff > 1%: ✓ (10%)
- **Confidence = (40 + 30 + 20 + 10) / 100 = 1.00**

## 3. Verify Operation Diagnostics

### 3.1 DeliveryAssessment Structure

```go
type DeliveryAssessment struct {
    State      string
    Confidence float64
    Evidence   FocusVerificationEvidence
    Messages   []string
}
```

### 3.2 Diagnostic Context Fields

| Field | Type | Description | Example Values |
|-------|------|-------------|----------------|
| `delivery_state` | string | Final delivery state | `verified`, `sent_unverified`, `unknown`, `failed` |
| `confidence` | string | Overall confidence score (2 decimal) | `"0.85"`, `"0.60"` |
| `messages` | string | Diagnostic messages (semicolon-separated) | `"Focus verified: confidence=0.85; Combined verification..."` |

### 3.3 Delivery State Values

| State | Confidence Range | Description |
|-------|------------------|-------------|
| `verified` | ≥ 0.8 | Message successfully delivered and verified |
| `sent_unverified` | 0.5 - 0.79 | Message sent but verification uncertain |
| `unknown` | < 0.5 | Verification failed or inconclusive |
| `failed` | N/A | Operation failed (error state) |

### 3.4 Overall Confidence Calculation

Overall confidence combines focus and message evidence:

**When message evidence is available:**
```
overall_confidence = (focus_confidence × 0.4) + (message_confidence × 0.6)
```

**When message evidence is unavailable (focus-only):**
```
overall_confidence = focus_confidence
```

## 4. Complete Diagnostic Flow Example

### 4.1 Focus Operation

**Input:** Focus on conversation "张三"

**Evidence Collected:**
- Node Still Exists: ✓
- Node Has Active State: ✓
- Title Contains Target: ✓
- Panel Switch Detected: ✗
- Message Area Visible: ✓

**Diagnostics Output:**
```json
{
  "locate_source": "tree_path_name",
  "node_still_exists": "true",
  "node_has_active_state": "true",
  "title_contains_target": "true",
  "panel_switch_detected": "false",
  "message_area_visible": "true",
  "evidence_count": "4",
  "confidence": "0.80"
}
```

### 4.2 Send Operation

**Input:** Send message "Hello World"

**Evidence Collected:**
- New Message Nodes: 1
- Message Node Added: ✓
- Message Content Match: ✓
- Screenshot Changed: ✓
- Chat Area Diff: 0.05

**Diagnostics Output:**
```json
{
  "new_message_nodes": "1",
  "message_node_added": "true",
  "message_content_match": "true",
  "screenshot_changed": "true",
  "chat_area_diff": "0.05",
  "confidence": "1.00"
}
```

### 4.3 Verify Operation

**Input:** Verify message "Hello World" delivery

**Diagnostics Output:**
```json
{
  "delivery_state": "verified",
  "confidence": "0.92",
  "messages": "Focus verified: confidence=0.80; Combined verification: confidence=0.92; Focus evidence: 4 items, confidence=0.80; Message evidence: new_nodes=1, confidence=1.00"
}
```

## 5. Implementation Notes

### 5.1 Conversion Functions

The following utility functions convert evidence structures to diagnostic context:

```go
// ConvertFocusEvidenceToDiagnostics
func ConvertFocusEvidenceToDiagnostics(evidence FocusVerificationEvidence) map[string]string

// ConvertMessageEvidenceToDiagnostics
func ConvertMessageEvidenceToDiagnostics(evidence SendVerificationEvidence) map[string]string

// ConvertDeliveryAssessmentToDiagnostics
func ConvertDeliveryAssessmentToDiagnostics(assessment DeliveryAssessment) map[string]string
```

### 5.2 Format Requirements

- **Confidence values**: Always formatted as 2 decimal places (e.g., `"0.85"`, `"1.00"`)
- **Boolean values**: Always lowercase strings (`"true"`, `"false"`)
- **Integer values**: Always string format (e.g., `"3"`, `"1"`)
- **Messages**: Semicolon-separated list for multiple messages

### 5.3 Mock vs Real Bridge Alignment

Both mock and real bridges should produce identical diagnostic field names and formats:

| Field | Mock Implementation | Real Implementation |
|-------|---------------------|---------------------|
| `locate_source` | ✓ | ✓ |
| `evidence_count` | ✓ | ✓ |
| `confidence` | ✓ | ✓ |
| `delivery_state` | ✓ | ✓ |
| `new_message_nodes` | ✓ | ✓ |
| `message_content_match` | ✓ | ✓ |

## 6. Testing

### 6.1 Test Coverage

The following test files verify diagnostics schema:

- `diagnostic_flow_test.go`: Complete chain testing with state-changing mock
- `adapter_diagnostic_test.go`: Minimum closed-loop diagnostic testing
- `rules_test.go`: Individual rule verification

### 6.2 Test Commands

```bash
# Run diagnostic flow tests
go test -v ./internal/agent/adapter/wechat -run "TestDiagnosticFlow" -timeout 30s

# Run minimum closed-loop tests
go test -v ./internal/agent/adapter/wechat -run "TestWeChatAdapter_MinimumClosedLoop" -timeout 30s

# Run all WeChat adapter tests
make test-adapter
```

## 7. Debug Commands

The `wechat-debug` tool provides commands to inspect diagnostics:

```bash
# Scan conversations with diagnostics
wechat-debug scan --json

# Focus on contact with diagnostics
wechat-debug focus --contact "张三" --json

# Send message with diagnostics
wechat-debug send --contact "张三" --message "Hello" --json

# Verify message delivery with diagnostics
wechat-debug verify --contact "张三" --message "Hello" --json

# Run complete chain with diagnostics
wechat-debug run-chain --contact "张三" --message "Test" --json
```

## 8. References

- `internal/agent/adapter/wechat/rules.go`: Evidence structures and conversion functions
- `internal/agent/adapter/wechat/diagnostic_flow_test.go`: State-changing mock implementation
- `internal/agent/adapter/wechat/adapter_diagnostic_test.go`: Minimum closed-loop tests
- `cmd/wechat-debug/main.go`: Debug command implementation
