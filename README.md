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
| Windows Bridge | Go + Windows Native API | ✅ 骨架完成 |
| Control Plane / Gateway | Go | ✅ 骨架完成 |
| LLM Orchestrator | Python | ✅ 骨架完成 |
| Admin Web | React/Next.js | 待后续 |

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

## 许可证

MIT License
