# WeChat Bridge Diagnosis Report - 2026-03-24

## Executive Summary
**Problem Classification: B类** – accessible 子树过浅/为空

## 真实执行命令及关键输出

### 1. `bridge-dump find-wechat`
```
Searching for WeChat windows...
Found 2 WeChat window(s):
  [1] Handle: 592004, Class: Chrome_WidgetWin_0, Title: 微信
  [2] Handle: 592008, Class: Qt51514QWindowIcon, Title: 微信
```

### 2. `bridge-dump debug-windows`
- Output: 29.3KB (列出451个顶层窗口)
- 无关键诊断字段显示

### 3. `bridge-dump debug-accessible 592004`
```
=== Debug: Accessible Diagnostics for Handle: 0x90884 (592004) ===
Window Information:
  Handle: 0x90884 (592004)
  Class: Chrome_WidgetWin_0
  Title: 微信

Attempting GetAccessible...
  SUCCESS: Accessible object obtained
  Diagnostics:
    - AccessibleObjectFromWindow succeeded
      return_code_hex: 0x0
      accessible_obtained: true
      objid_client: 0xFFFFFFFC
      window_handle: 592004
      pAcc_is_nil: false
      child_count: 0
      iid_accessible: {1636251360 15421 4559 [129 12 0 170 0 56 155 113]}
      return_code: 0x0
```

### 4. `bridge-dump debug-nodes 592004 10`
关键诊断字段：
```
- accessible_obtained: true
- root_child_count: 0
- children_enumerated: 0
- total_nodes_count: 1
- fallback_used: false
- bridge_layer_status: success
- effective_window_handle: 592004
- diagnostic_summary: Got accessible subtree with 1 total nodes, root child count: 0
```

### 5. `wechat-debug find-window`
```
Found 2 WeChat instance(s):
  [1] AppID: wechat, InstanceID: 微信
  [2] AppID: wechat, InstanceID: 微信
```

### 6. `wechat-debug scan`
```
Scan completed:
  Conversations found: 0

Diagnostics:
  window_class: Chrome_WidgetWin_0
  window_title: 微信
  candidates_found: 0
  locate_source: unknown
  evidence_count: 0
  new_message_nodes: 0
  confidence: 0.00
  window_handle: 592004
  hits_found: 0
  message_content_match: false
  delivery_state: unknown
  nodes_found: 1
```

## 问题归类：B类 (accessible 子树过浅/为空)

### 归类依据：
1. **accessible_obtained: true** – 桥接层成功获取可访问对象
2. **child_count: 0** – 根节点子节点数为0
3. **total_nodes_count: 1** – 整个子树仅有根节点（无任何子节点）
4. **fallback_used: false** – 未触发回退机制
5. **bridge_layer_status: success** – 桥接层状态正常
6. **effective_window_handle** 仍为原窗口句柄（592004），未切换到子窗口
7. **wechat-debug scan** 扫描到 `nodes_found: 1` 但 `candidates_found: 0`，证实无会话节点可匹配

### 排除其他类别：
- **A类排除**：`accessible_obtained=true`，`bridge_layer_blocked` 未出现（应为false）
- **C类排除**：`total_nodes_count=1` 过少，无法进行规则匹配

## 根本原因分析
当前 `bridge.go` 的 `EnumerateAccessibleNodes` 函数仅在 `GetAccessible` 失败时才调用 `tryChildWindows`。但在真实微信环境中：
1. 父窗口（Chrome_WidgetWin_0）的 `AccessibleObjectFromWindow` 成功
2. 但返回的可访问对象 `child_count=0`，子树为空
3. 真实的会话控件位于某个子窗口（或孙窗口）的可访问子树中
4. 由于父窗口的 `GetAccessible` 成功，代码未尝试子窗口，直接返回仅有根节点的树

## 下一步最小修复点

### 需要修改的文件（仅1个）：
**[internal/agent/windows/bridge.go](internal/agent/windows/bridge.go)**

### 具体修改位置：
**函数：`EnumerateAccessibleNodes`（约第606行）**

**修改逻辑**：
```go
// 当前逻辑：
pAcc, result := b.GetAccessible(windowHandle)
if result.Status != adapter.StatusSuccess {
    // 只在失败时尝试子窗口
    childHandle, childAcc, childResult := b.tryChildWindows(windowHandle)
    // ...
}

// 修改后逻辑：
pAcc, result := b.GetAccessible(windowHandle)
childHandle := windowHandle  // 默认为原窗口句柄

if result.Status != adapter.StatusSuccess {
    // 失败时尝试子窗口
    childHandle, pAcc, result = b.tryChildWindows(windowHandle)
} else {
    // 成功但无子节点时也尝试子窗口
    childCount := b.getAccChildCount(pAcc)
    if childCount == 0 {
        // 父窗口可访问但子树为空，尝试子窗口
        foundChildHandle, foundChildAcc, childResult := b.tryChildWindows(windowHandle)
        if childResult.Status == adapter.StatusSuccess && foundChildAcc != nil {
            childHandle = foundChildHandle
            pAcc = foundChildAcc
            result = childResult
            // 更新effective_window_handle诊断信息
        }
    }
}
```

### 可选增强（同一文件）：
- 在 `tryChildWindows` 中，若子窗口的 `childCount` 也为0，可尝试更深的孙窗口或不同的 `OBJID`（如 `OBJID_WINDOW` 等）

## 暂时不要改的文件（避免扩散修改）

1. `internal/agent/adapter/wechat/adapter.go` – 规则层无需改动
2. `internal/agent/adapter/wechat/rules.go` – 规则层无需改动
3. `cmd/wechat-debug/main.go` – 诊断工具已输出足够信息
4. `cmd/bridge-dump/main.go` – 诊断工具功能完整
5. 任何测试文件或配置文件

## 预期修复效果
修复后，`EnumerateAccessibleNodes` 应能：
1. 检测到父窗口 `child_count=0` 的情况
2. 自动降级尝试子窗口
3. 找到承载真实控件的子窗口句柄
4. 返回包含会话列表节点的完整 accessible 子树
5. `wechat-debug scan` 应能找到候选会话节点（C类规则层匹配）

## 验证方法
修复后重新执行：
```bash
./bridge-dump debug-nodes <微信窗口句柄> 10
./wechat-debug scan
```
预期看到 `total_nodes_count > 1` 且 `candidates_found > 0`。

---
**诊断执行时间：** 2026-03-24
**环境：** Windows 11 + 真实微信客户端
**桥接工具版本：** bridge-dump.exe (build: 2026-03-24)