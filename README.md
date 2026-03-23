# 跨平台自动客服桌面代理系统

## 项目简介

这是一个跨平台自动客服桌面代理系统，能够在客户电脑上安装并自动处理聊天消息。

### 第一版范围约束

| 约束项 | 说明 |
|--------|------|
| 平台 | 仅 Windows 10/11 |
| 聊天软件 | 仅微信桌面版 |
| 消息类型 | 仅单聊文本消息 |
| 功能闭环 | 识别 → 上报 → AI 回复 → 发送 → 验证 |

## 技术栈

| 组件 | 技术栈 | 状态 |
|------|--------|------|
| Agent Core | Go | ✅ 骨架完成 |
| Windows Bridge | Go + Windows Native API | ✅ IAccessible 实现完成 |
| Control Plane / Gateway | Go | ✅ 骨架完成 |
| LLM Orchestrator | Python | ✅ 骨架完成 |
| Admin Web | React/Next.js | 待后续 |

## Windows Bridge 实现详情

### IAccessible 节点遍历

Windows Bridge 已实现完整的 IAccessible 接口遍历功能：

```go
// internal/agent/windows/bridge.go
func (b *Bridge) EnumerateAccessibleNodes(windowHandle uintptr) ([]AccessibleNode, adapter.Result)
```

**功能特性：**
- ✅ 递归遍历 IAccessible 子节点
- ✅ 提取节点角色 (Role)、名称 (Name)、类名 (ClassName)
- ✅ 获取节点边界框 (Bounds)
- ✅ 支持深度限制避免无限递归

### 会话切换实现

WeChat 适配器已实现真实的会话切换功能：

```go
// internal/agent/adapter/wechat/adapter.go
func (a *WeChatAdapter) Focus(conv protocol.ConversationRef) adapter.Result
```

**实现步骤：**
1. 聚焦到微信窗口
2. 根据 ListPosition 计算点击坐标
3. 点击目标会话项
4. 返回置信度评分

### 诊断工具

新增 `bridge-dump` 命令行工具用于调试：

```bash
# 查找微信窗口
bridge-dump find-wechat

# 获取窗口信息
bridge-dump window-info <handle>

# 列出可访问性节点
bridge-dump list-nodes <handle> --json --depth 3

# 聚焦窗口
bridge-dump focus <handle>
```

**支持选项：**
- `--json`: JSON 格式输出
- `--depth <n>`: 递归深度限制（默认 5）

## 目录结构

```
auto-customer-service/
├── README.md
├── go.mod                          # Go 模块定义
├── go.sum
├── .gitignore
│
├── cmd/                            # 可执行程序
│   ├── agent/                      # Agent 主程序
│   │   └── main.go
│   ├── gateway/                    # Gateway 主程序 (Go)
│   │   └── main.go
│   └── mock-chat/                  # Mock Chat App (Go)
│       └── main.go
│
├── pkg/                            # Go 包 (Agent Core + Windows Bridge)
│   ├── agent/                      # Agent Core
│   │   ├── adapter/                # 适配器层
│   │   │   ├── interface.go        # ChatAdapter 接口
│   │   │   ├── wechat/             # 微信适配器
│   │   │   │   └── adapter.go
│   │   │   └── manager.go          # 适配器管理器
│   │   ├── state/                  # 状态管理
│   │   │   ├── statemachine.go     # 状态机
│   │   │   ├── conversation.go     # 会话状态
│   │   │   └── review.go           # 审核状态
│   │   ├── comm/                   # 通信模块
│   │   │   ├── client.go           # WebSocket 客户端
│   │   │   └── protocol.go         # 协议定义
│   │   ├── task/                   # 任务调度
│   │   │   ├── dispatcher.go       # 任务分发
│   │   │   └── executor.go         # 任务执行
│   │   ├── idempotency/            # 幂等存储
│   │   │   ├── store.go            # 存储接口
│   │   │   └── memory.go           # 内存实现
│   │   └── agent.go                # Agent 主逻辑
│   │
│   ├── windows/                    # Windows Bridge
│   │   ├── bridge.go               # 桥接接口
│   │   ├── uia/                    # UIA 实现
│   │   │   └── uia.go
│   │   └── ocr/                    # OCR 实现
│   │       └── ocr.go
│   │
│   └── mock/                       # Mock 工具
│       ├── chatapp/                # Mock Chat App
│       │   ├── app.go              # 应用实现
│       │   ├── window.go           # 窗口管理
│       │   └── ui.go               # UI 组件
│
├── gateway/                        # Gateway (Server) - Go
│   ├── cmd/                        # 可执行程序
│   │   └── main.go
│   ├── internal/                   # 内部包
│   │   ├── server/                 # HTTP/WebSocket 服务器
│   │   │   └── server.go
│   │   ├── llm/                    # LLM 编排
│   │   │   └── orchestrator.go
│   │   └── protocol/               # 协议定义
│   │       └── message.go
│   └── go.mod                      # Gateway 独立模块
│
├── llm-orchestrator/               # LLM Orchestrator (Python)
│   ├── main.py                     # 主程序
│   ├── requirements.txt            # 依赖
│   ├── orchestrator/               # 编排逻辑
│   │   ├── __init__.py
│   │   ├── llm.py                  # LLM 调用
│   │   └── prompt.py               # 提示词工程
│   └── config.yaml                 # 配置文件
│
├── web/                            # Admin Web (React/Next.js)
│   ├── app/                        # Next.js 应用
│   │   ├── page.tsx
│   │   └── layout.tsx
│   ├── components/                 # React 组件
│   └── lib/                        # 工具库
│
├── tests/                          # 测试
│   ├── unit/                       # 单元测试
│   ├── integration/                # 集成测试
│   └── e2e/                        # E2E 测试
```

## 快速开始

### 1. 初始化 Go 模块

```bash
go mod init github.com/yourorg/auto-customer-service
go mod tidy
```

### 2. 运行测试

使用 Makefile:

```bash
# 运行单元测试（快速，无外部依赖）
make test-unit

# 运行集成测试
make test-integration

# 运行网关测试
make test-gateway

# 运行所有测试
make test-all
```

或使用 Task (推荐):

```bash
# 运行单元测试
task test-unit

# 运行集成测试
task test-integration

# 运行网关测试
task test-gateway

# 运行所有测试
task test-all
```

### 3. 运行 Mock Chat App

```bash
# 使用 Makefile
make run-mock-chat

# 或使用 Task
task run-mock-chat

# 或直接运行
go run ./cmd/mock-chat/main.go
```

### 4. 运行 Gateway 服务器

```bash
# 使用 Makefile
make run-gateway

# 或使用 Task
task run-gateway

# 或直接运行
go run ./cmd/gateway/main.go
```

### 5. 构建所有包

```bash
# 使用 Makefile
make build

# 或使用 Task
task build

# 或直接运行
go build ./...
```

## 开发文档

查看开发日志目录获取详细设计文档：

- [01-工程化设计方案.md](开发日志/01-工程化设计方案.md)
- [02-工程约束设计稿.md](开发日志/02-工程约束设计稿.md)
- [03-工程骨架设计.md](开发日志/03-工程骨架设计.md)
- [04-骨架代码设计.md](开发日志/04-骨架代码设计.md)

### 已完成工作

#### Step 1: IAccessible 节点遍历实现
- ✅ 实现 `EnumerateAccessibleNodes()` 递归遍历 IAccessible 子节点
- ✅ 提取节点角色、名称、类名、边界框信息
- ✅ 支持深度限制避免无限递归

#### Step 2: bridge-dump 工具升级
- ✅ 添加 `--json` 选项支持 JSON 格式输出
- ✅ 添加 `--depth` 选项支持递归深度限制
- ✅ 支持 `find-wechat`、`window-info`、`list-nodes`、`focus` 命令

#### Step 3: WeChat 适配器会话切换
- ✅ 改进 `Focus()` 方法实现真实会话切换
- ✅ 根据 ListPosition 计算点击坐标
- ✅ 返回置信度评分

#### Step 4: 发送链路改进
- ✅ 改进 `Send()` 方法阶段式确认
- ✅ 添加详细诊断信息
- ✅ 改进 `Verify()` 方法验证逻辑

#### Step 5: 任务状态机改进
- ✅ 添加 `task.progress` 事件发送
- ✅ 实现 6 个进度阶段 (detecting, scanning, finding, focusing, sending, verifying)
- ✅ 添加 `sendProgress()` 辅助函数

#### Step 6: 端到端测试
- ✅ 新增 Gateway-Agent E2E 测试套件
- ✅ 测试 WebSocket 连接、消息发送/接收
- ✅ 测试任务进度流、完成流、失败流
- ✅ 测试多命令并发处理

#### Step 7: README 文档完善
- ✅ 更新技术栈状态
- ✅ 添加 Windows Bridge 实现详情
- ✅ 添加任务状态机说明
- ✅ 添加端到端测试说明
- ✅ 完善协议说明文档

## 任务状态机

### 进度阶段

Agent 在执行任务时会发送 `task.progress` 事件，包含以下阶段：

| 阶段 (Stage) | 描述 |
|--------------|------|
| `detecting` | 检测应用实例中 |
| `scanning` | 扫描会话列表中 |
| `finding` | 查找目标会话中 |
| `focusing` | 切换到目标会话 |
| `sending` | 发送消息中 |
| `verifying` | 验证消息发送中 |

### 任务状态流转

```
pending → sending → sent_unverified → verified / unknown_delivery_state / failed
```

## 端到端测试

### Gateway ↔ Agent 通信测试

新增完整的端到端测试，验证 WebSocket 通信：

```bash
# 运行 Gateway-Agent E2E 测试
go test -v ./tests/integration/ -run "TestGatewayAgent_E2E"
```

**测试覆盖：**
- ✅ WebSocket 连接建立
- ✅ Agent 发送事件到 Gateway
- ✅ Gateway 发送命令到 Agent
- ✅ 任务进度流 (task.progress)
- ✅ 任务完成流 (task.completed)
- ✅ 任务失败流 (task.failed)
- ✅ 多命令并发处理

## 协议说明

### 消息方向

- **Command**: Server → Agent (下行)
- **Event**: Agent → Server (上行)
- **Audit**: Agent → Server (上行)

### 核心数据结构

- **ConversationRef**: 运行时会话引用
- **ConversationIdentity**: 逻辑会话身份
- **MessageObs**: 观测消息模型
- **Result**: 统一返回对象

### 事件类型

| 事件类型 | 载荷类型 | 方向 | 描述 |
|----------|----------|------|------|
| `conversation.new_message` | NewMessagePayload | Agent → Gateway | 新消息通知 |
| `reply.execute` | ReplyExecutePayload | Gateway → Agent | 执行回复命令 |
| `conversation.mode.set` | ConvModeSetPayload | Gateway → Agent | 设置会话模式 |
| `diagnostic.capture` | DiagnosticCapturePayload | Gateway → Agent | 捕获诊断信息 |
| `task.progress` | TaskProgressPayload | Agent → Gateway | 任务进度更新 |
| `task.completed` | TaskCompletedPayload | Agent → Gateway | 任务完成通知 |
| `task.failed` | TaskFailedPayload | Agent → Gateway | 任务失败通知 |

## 许可证

MIT License
