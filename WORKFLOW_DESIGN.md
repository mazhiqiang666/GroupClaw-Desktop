# 主程序工作流设计

## 概述
本文档描述自动客服桌面代理的主循环工作流设计。工作流从被动响应式（WebSocket命令驱动）转变为主动监控式（自主检测、处理、回复）。

## 设计原则
1. **主动监控**：持续监测联系人列表，发现新消息
2. **会话状态管理**：维护每个联系人的会话状态和消息历史
3. **异步处理**：消息检测、AI回复、发送验证异步进行
4. **容错恢复**：每个步骤都有降级和重试机制
5. **状态持久化**：会话状态持久化存储，支持重启恢复

## 主循环步骤

### 1. 监测联系人列表
**目标**：定期检测微信左侧联系人列表，识别有新消息的联系人
**实现**：
```go
// 使用视觉/OCR检测左侧栏
result := adapter.DetectConversations(windowHandle)
// 或者使用 accessibility 检测
nodes := bridge.EnumerateAccessibleNodes(windowHandle)
contacts := filterContacts(nodes)
```
**输出**：
- 当前可见的联系人列表
- 每个联系人的未读消息状态（红点、未读计数）
- 联系人矩形位置（用于点击）

### 2. 打开有新消息的联系人
**目标**：点击有未读消息的联系人，打开聊天窗口
**实现**：
```go
// 如果联系人已在左侧列表可见，直接点击
clickResult := bridge.Click(contactRect)
// 否则使用搜索框导航流程
searchResult := adapter.NavigateToContact(contactName)
```
**验证**：
- 检查顶部标题是否显示联系人名称
- 检查左侧联系人是否高亮
- 检查消息区域是否可见

### 3. 读取新增消息
**目标**：读取当前聊天窗口中的新消息（上次读取之后的）
**实现**：
```go
// 读取最新消息
messages, result := adapter.Read(convRef, limit)
// 过滤出新增消息（对比session历史）
newMessages := filterNewMessages(messages, session.LastMessageID)
```
**输出**：
- 新增消息列表（发送者、内容、时间戳）
- 消息指纹（用于去重）

### 4. 更新该联系人的session
**目标**：更新会话状态，记录已读取的消息
**实现**：
```go
session := sessionManager.GetOrCreate(contactID)
session.LastReadTime = time.Now()
session.LastMessageID = latestMessage.ID
session.MessageHistory = append(session.MessageHistory, newMessages...)
sessionManager.Save(session)
```

### 5. 调用远端agent获取回复
**目标**：将消息上下文发送给AI agent，获取回复内容
**实现**：
```go
// 构建请求上下文
context := BuildAgentContext(session.MessageHistory, contactInfo)
// 调用远端API
reply, err := callRemoteAgent(context)
// 处理响应
if err == nil && reply.Valid {
    session.PendingReply = reply.Content
}
```
**注意**：
- 异步调用，设置超时
- 支持失败重试
- 记录调用状态

### 6. 确认当前聊天框仍属于该联系人
**目标**：在发送前确认仍在正确的聊天窗口
**实现**：
```go
// 验证当前聊天窗口
isCorrectChat := verifyCurrentChat(contactName)
if !isCorrectChat {
    // 重新聚焦到目标联系人
    focusResult := adapter.Focus(convRef)
}
```
**验证信号**：
- 顶部标题匹配
- 左侧联系人高亮状态
- 消息区域可见性

### 7. 输入并发送回复
**目标**：将回复内容输入到聊天框并发送
**实现**：
```go
// 定位并激活输入框
inputBoxResult := adapter.LocateInputBox(convRef)
// 注入回复文本（优先使用Ctrl+V策略）
injectResult := adapter.InjectReplyText(windowHandle, replyContent, "ctrl_v")
// 发送消息（Enter键）
sendResult := adapter.Send(convRef, replyContent, taskID)
```
**验证**：
- 输入框激活状态
- 文本注入成功验证
- 发送操作确认

### 8. 更新session
**目标**：记录发送的回复，更新会话状态
**实现**：
```go
session.LastReplyTime = time.Now()
session.LastReplyContent = replyContent
session.PendingReply = ""
session.ReplyHistory = append(session.ReplyHistory, replyRecord)
sessionManager.Save(session)
```

## 架构组件

### SessionManager
```go
type SessionManager struct {
    mu sync.RWMutex
    sessions map[string]*ChatSession
    store    SessionStore
}

type ChatSession struct {
    ContactID       string
    LastReadTime    time.Time
    LastMessageID   string
    MessageHistory  []Message
    LastReplyTime   time.Time
    LastReplyContent string
    ReplyHistory    []ReplyRecord
    PendingReply    string
    UnreadCount     int
    IsActive        bool
}
```

### MonitorService
```go
type MonitorService struct {
    adapter      adapter.ChatAdapter
    sessionMgr   *SessionManager
    agentClient  *RemoteAgentClient
    pollInterval time.Duration
    windowHandle uintptr
}

func (m *MonitorService) Start() {
    for {
        m.monitorCycle()
        time.Sleep(m.pollInterval)
    }
}

func (m *MonitorService) monitorCycle() {
    // 实现8个步骤的主循环
}
```

### RemoteAgentClient
```go
type RemoteAgentClient struct {
    gatewayAddr string
    timeout     time.Duration
}

func (c *RemoteAgentClient) GetReply(context AgentContext) (string, error) {
    // 调用远端Gateway/LLM服务
}
```

## 集成到现有系统

### 方案A：独立监控进程
创建新的可执行文件 `cmd/monitor/main.go`，独立于现有的WebSocket agent运行。

**优点**：
- 与现有系统解耦
- 独立部署和扩展
- 不影响现有命令响应

**缺点**：
- 需要单独管理生命周期
- 状态同步可能复杂

### 方案B：扩展现有agent
在现有 `cmd/agent/main.go` 中添加监控模式，通过命令行参数或环境变量启用。

**优点**：
- 代码复用率高
- 统一状态管理
- 简化部署

**缺点**：
- 可能增加复杂性
- 与命令响应模式可能有冲突

### 推荐方案：方案B（扩展现有agent）
1. 添加 `--monitor` 命令行标志启用监控模式
2. 监控模式和WebSocket模式可以共存或互斥
3. 共享SessionManager和适配器实例

## 错误处理和恢复

### 降级策略
1. **视觉检测失败** → 降级到accessibility检测
2. **OCR失败** → 降级到几何布局分析
3. **远程agent失败** → 使用本地模板回复或跳过
4. **发送失败** → 重试（最多3次）后标记为失败

### 状态恢复
1. **窗口失去焦点** → 重新获取窗口句柄
2. **会话切换意外** → 重新验证并聚焦
3. **输入框定位失败** → 尝试备用定位策略

## 配置参数

```yaml
monitor:
  poll_interval: 5000  # 监测间隔（毫秒）
  max_retries: 3       # 最大重试次数
  timeout_ms: 10000    # 操作超时时间

session:
  max_history: 100     # 最大消息历史记录数
  persistence_file: "./sessions.json"

agent:
  endpoint: "http://localhost:8080/api/reply"
  timeout_ms: 30000
```

## 实现状态

✅ **已完成所有阶段**

1. **✅ 阶段1**：创建SessionManager和SessionStore
   - `internal/agent/session/manager.go` - 完整的会话管理实现
   - 支持内存存储和文件存储
   - 线程安全的会话操作

2. **✅ 阶段2**：实现MonitorService骨架和主循环
   - `internal/agent/monitor/service.go` - 完整的8步骤监控循环
   - 支持远程agent客户端集成
   - 完整的错误处理和恢复机制

3. **✅ 阶段3**：集成到现有agent（添加--monitor标志）
   - `cmd/agent/main.go` - 扩展支持`--monitor`命令行标志
   - 同时支持WebSocket命令模式和自主监控模式
   - 共享SessionManager实例

4. **✅ 阶段4**：实现各步骤的具体逻辑
   - 步骤1：监测联系人列表（视觉/accessibility检测）
   - 步骤2：打开有新消息的联系人（导航逻辑）
   - 步骤3：读取新增消息（去重和过滤）
   - 步骤4：更新会话状态
   - 步骤5：调用远端agent获取回复
   - 步骤6：确认当前聊天框仍属于该联系人
   - 步骤7：输入并发送回复（使用优化策略）
   - 步骤8：更新会话回复记录

5. **✅ 阶段5**：添加配置和错误处理
   - 可配置的轮询间隔、重试次数、超时时间
   - 降级策略：视觉→accessibility→几何分析
   - 状态恢复机制

6. **🔧 阶段6**：测试和调试
   - 编译通过，基础架构完成
   - 需要在实际微信环境中进行端到端测试

## 相关文件
- `cmd/agent/main.go` - 主程序入口
- `internal/agent/session/manager.go` - 会话管理
- `internal/agent/monitor/service.go` - 监控服务
- `internal/agent/remote/client.go` - 远端agent客户端

## 使用监控模式

### 启动命令

```bash
# 启动agent并启用监控模式（5秒轮询间隔）
./agent --monitor --poll-interval 5s

# 使用模拟agent进行测试
./agent --monitor --mock-agent

# 指定远端agent端点
./agent --monitor --agent-endpoint "http://localhost:8080/api/reply"
```

### 命令行参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--monitor` | `false` | 启用监控模式 |
| `--poll-interval` | `5s` | 监控轮询间隔 |
| `--agent-endpoint` | `http://localhost:8080/api/reply` | 远端agent端点 |
| `--mock-agent` | `false` | 使用模拟agent（测试用） |

### 环境变量

```bash
# 覆盖Gateway地址
export GATEWAY_ADDR="localhost:8080"

# 同时支持WebSocket连接和监控模式
```

### 监控流程示例

1. **启动监控**：
   ```bash
   ./agent --monitor --poll-interval 10s
   ```

2. **日志输出**：
   ```
   监控服务启动成功
   检测到 15 个联系人
   发现 2 个有未读消息的联系人
   处理联系人: 张三 (未读: 3)
   步骤1: 监测联系人列表
   步骤2: 打开联系人张三
   ...
   ```

3. **会话管理**：
   - 会话状态自动持久化
   - 消息去重和过滤
   - 回复记录跟踪

### 测试建议

1. **使用模拟agent**：
   ```bash
   ./agent --monitor --mock-agent --poll-interval 30s
   ```

2. **验证基础流程**：
   - 确保微信窗口可见
   - 监控服务能检测到联系人列表
   - 会话管理器能正确创建和更新会话

3. **端到端测试**：
   - 实际收到微信消息
   - 监控服务自动检测并处理
   - 验证回复发送成功
