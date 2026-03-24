# WeChat Debugging 调试指南

## Real 模式联调 SOP

### 1. 测试原则

**先 Mock 后 Real** 原则：
1. **第一阶段：Mock 回归测试** - 确保所有功能在模拟环境中正常工作
2. **第二阶段：Real 手工冒烟测试** - 在真实 WeChat 环境中验证关键流程

### 2. Mock 模式回归测试（第一步）

在进入 Real 模式前，确保 Mock 模式所有测试通过：

```bash
# 1. 运行 Mock 模式完整链路测试
wechat-debug run-chain --contact "测试联系人" --message "测试消息" --mock --json

# 2. 验证 Mock 模式各步骤
wechat-debug scan --mock --json
wechat-debug focus --contact "测试联系人" --mock --json
wechat-debug send --contact "测试联系人" --message "测试消息" --mock --json
wechat-debug verify --contact "测试联系人" --message "测试消息" --mock --json
```

**验证标准：**
- ✅ 所有命令返回状态为 `StatusSuccess`
- ✅ JSON 输出结构符合预期
- ✅ Confidence 置信度在合理范围内 (≥ 0.8)
- ✅ Diagnostics 诊断信息完整

### 3. Real 模式手工冒烟测试（第二步）

在 Mock 测试通过后，进行 Real 模式测试：

**前提条件：**
- WeChat 桌面版正在运行
- 至少有一个测试用的联系人会话
- 确保测试不会发送垃圾消息给真实联系人

#### 3.1 基础扫描测试

```bash
# 1. 查找 WeChat 窗口
wechat-debug find-window --json

# 2. 扫描会话列表
wechat-debug scan --json

# 3. 聚焦到目标联系人
wechat-debug focus --contact "<真实联系人姓名>" --json

# 4. 发送测试消息
wechat-debug send --contact "<真实联系人姓名>" --message "自动化测试消息，请忽略" --json

# 5. 验证消息发送
wechat-debug verify --contact "<真实联系人姓名>" --message "自动化测试消息，请忽略" --json
```

#### 3.2 完整链路测试

```bash
# 运行完整链路：scan → focus → send → verify
wechat-debug run-chain --contact "<真实联系人姓名>" --message "自动化测试消息，请忽略" --json
```

### 4. Mock / Real Baseline 输出对比

#### 4.1 Mock run-chain JSON 样例

```json
{
  "mode": "chain",
  "steps": [
    {
      "step": "detect",
      "status": "StatusSuccess",
      "confidence": 1.0,
      "instances": [
        {
          "app_id": "wechat",
          "instance_id": "mock-12345",
          "window_handle": "12345"
        }
      ],
      "diagnostics": [
        {
          "timestamp": "2026-03-24T10:30:00Z",
          "level": "info",
          "message": "Mock WeChat instance detected",
          "context": {
            "instance_count": "1",
            "detect_method": "mock"
          }
        }
      ]
    },
    {
      "step": "scan",
      "status": "StatusSuccess",
      "confidence": 0.95,
      "conversations": [
        {
          "display_name": "张三",
          "list_position": 0,
          "list_neighborhood_hint": ["list item 1"],
          "bounds": {
            "x": 100,
            "y": 200,
            "width": 300,
            "height": 50
          }
        },
        {
          "display_name": "李四",
          "list_position": 1,
          "list_neighborhood_hint": ["list item 2"],
          "bounds": {
            "x": 100,
            "y": 250,
            "width": 300,
            "height": 50
          }
        }
      ],
      "diagnostics": [
        {
          "timestamp": "2026-03-24T10:30:01Z",
          "level": "info",
          "message": "3 conversations found",
          "context": {
            "total_conversations": "3",
            "scan_method": "mock"
          }
        }
      ]
    },
    {
      "step": "focus",
      "status": "StatusSuccess",
      "confidence": 0.9,
      "conversation": {
        "display_name": "张三",
        "list_position": 0,
        "list_neighborhood_hint": ["list item 1"],
        "bounds": {
          "x": 100,
          "y": 200,
          "width": 300,
          "height": 50
        }
      },
      "diagnostics": [
        {
          "timestamp": "2026-03-24T10:30:02Z",
          "level": "info",
          "message": "Focus completed",
          "context": {
            "focus_method": "click",
            "locate_source": "bounds_center",
            "click_coordinates": "250,225"
          }
        }
      ]
    },
    {
      "step": "send",
      "status": "StatusSuccess",
      "confidence": 0.85,
      "conversation": {
        "display_name": "张三",
        "list_position": 0,
        "list_neighborhood_hint": ["list item 1"],
        "bounds": {
          "x": 100,
          "y": 200,
          "width": 300,
          "height": 50
        }
      },
      "diagnostics": [
        {
          "timestamp": "2026-03-24T10:30:03Z",
          "level": "info",
          "message": "Message sent successfully",
          "context": {
            "delivery_state": "sent_unverified",
            "message_length": "24",
            "send_method": "keyboard_input"
          }
        }
      ]
    },
    {
      "step": "verify",
      "status": "StatusSuccess",
      "confidence": 0.8,
      "conversation": {
        "display_name": "张三",
        "list_position": 0,
        "list_neighborhood_hint": ["list item 1"],
        "bounds": {
          "x": 100,
          "y": 200,
          "width": 300,
          "height": 50
        }
      },
      "message": {
        "content": "自动化测试消息，请忽略",
        "timestamp": "2026-03-24T10:30:04Z",
        "delivery_state": "verified"
      },
      "diagnostics": [
        {
          "timestamp": "2026-03-24T10:30:04Z",
          "level": "info",
          "message": "Message verified successfully",
          "context": {
            "delivery_state": "verified",
            "verification_method": "text_match",
            "match_confidence": "0.95"
          }
        }
      ]
    }
  ],
  "final": {
    "step": "verify",
    "status": "StatusSuccess",
    "confidence": 0.8,
    "conversation": {
      "display_name": "张三",
      "list_position": 0,
      "list_neighborhood_hint": ["list item 1"],
      "bounds": {
        "x": 100,
        "y": 200,
        "width": 300,
        "height": 50
      }
    },
    "message": {
      "content": "自动化测试消息，请忽略",
      "timestamp": "2026-03-24T10:30:04Z",
      "delivery_state": "verified"
    },
    "diagnostics": [
      {
        "timestamp": "2026-03-24T10:30:04Z",
        "level": "info",
        "message": "Message verified successfully",
        "context": {
          "delivery_state": "verified",
          "verification_method": "text_match",
          "match_confidence": "0.95"
        }
      }
    ]
  }
}
```

#### 4.2 Real scan JSON 样例

```json
{
  "mode": "single",
  "steps": [
    {
      "step": "scan",
      "status": "StatusSuccess",
      "confidence": 0.92,
      "conversations": [
        {
          "display_name": "张三",
          "list_position": 0,
          "list_neighborhood_hint": ["list item 1"],
          "bounds": {
            "x": 150,
            "y": 180,
            "width": 280,
            "height": 45
          }
        },
        {
          "display_name": "李四",
          "list_position": 1,
          "list_neighborhood_hint": ["list item 2"],
          "bounds": {
            "x": 150,
            "y": 225,
            "width": 280,
            "height": 45
          }
        }
      ],
      "diagnostics": [
        {
          "timestamp": "2026-03-24T10:31:00Z",
          "level": "info",
          "message": "2 conversations found",
          "context": {
            "total_conversations": "2",
            "scan_method": "iaccessible",
            "window_handle": "0x000A1234",
            "scan_duration_ms": "125"
          }
        }
      ]
    }
  ],
  "final": {
    "step": "scan",
    "status": "StatusSuccess",
    "confidence": 0.92,
    "conversations": [
      {
        "display_name": "张三",
        "list_position": 0,
        "list_neighborhood_hint": ["list item 1"],
        "bounds": {
          "x": 150,
          "y": 180,
          "width": 280,
          "height": 45
        }
      },
      {
        "display_name": "李四",
        "list_position": 1,
        "list_neighborhood_hint": ["list item 2"],
        "bounds": {
          "x": 150,
          "y": 225,
          "width": 280,
          "height": 45
        }
      }
    ],
    "diagnostics": [
      {
        "timestamp": "2026-03-24T10:31:00Z",
        "level": "info",
        "message": "2 conversations found",
        "context": {
          "total_conversations": "2",
          "scan_method": "iaccessible",
          "window_handle": "0x000A1234",
          "scan_duration_ms": "125"
        }
      }
    ]
  }
}
```

#### 4.3 Real focus JSON 样例

```json
{
  "mode": "single",
  "steps": [
    {
      "step": "focus",
      "status": "StatusSuccess",
      "confidence": 0.88,
      "conversation": {
        "display_name": "张三",
        "list_position": 0,
        "list_neighborhood_hint": ["list item 1"],
        "bounds": {
          "x": 150,
          "y": 180,
          "width": 280,
          "height": 45
        }
      },
      "diagnostics": [
        {
          "timestamp": "2026-03-24T10:32:00Z",
          "level": "info",
          "message": "Focus completed",
          "context": {
            "focus_method": "click",
            "locate_source": "bounds_center",
            "click_coordinates": "290,202",
            "click_delay_ms": "100",
            "window_handle": "0x000A1234"
          }
        },
        {
          "timestamp": "2026-03-24T10:32:01Z",
          "level": "info",
          "message": "Post-focus verification",
          "context": {
            "verification_method": "title_check",
            "title_match": "true",
            "post_focus_delay_ms": "200"
          }
        }
      ]
    }
  ],
  "final": {
    "step": "focus",
    "status": "StatusSuccess",
    "confidence": 0.88,
    "conversation": {
      "display_name": "张三",
      "list_position": 0,
      "list_neighborhood_hint": ["list item 1"],
      "bounds": {
        "x": 150,
        "y": 180,
        "width": 280,
        "height": 45
      }
    },
    "diagnostics": [
      {
        "timestamp": "2026-03-24T10:32:00Z",
        "level": "info",
        "message": "Focus completed",
        "context": {
          "focus_method": "click",
          "locate_source": "bounds_center",
          "click_coordinates": "290,202",
          "click_delay_ms": "100",
          "window_handle": "0x000A1234"
        }
      },
      {
        "timestamp": "2026-03-24T10:32:01Z",
        "level": "info",
        "message": "Post-focus verification",
        "context": {
          "verification_method": "title_check",
          "title_match": "true",
          "post_focus_delay_ms": "200"
        }
      }
    ]
  }
}
```

### 5. 字段稳定性说明

#### 5.1 必须稳定的字段（一致性要求）
- `mode`: 必须为 `"single"` 或 `"chain"`
- `step`: 步骤名称必须准确对应
- `status`: 状态值必须为有效的 `adapter.Status` 枚举值
- `confidence`: 置信度必须为 0-1 之间的浮点数，保留2位小数
- `diagnostics[].timestamp`: ISO 8601 格式时间戳
- `diagnostics[].level`: 必须为 `"info"`, `"warning"`, `"error"` 之一
- `conversations[].display_name`: 联系人显示名称
- `conversations[].list_position`: 列表位置（从0开始）

#### 5.2 允许变化的字段（环境相关）
- `instances[].instance_id`: Real 模式下为真实窗口句柄，Mock 模式下为模拟值
- `conversations[].bounds`: 坐标值因窗口位置和大小而变化
- `conversations[].list_neighborhood_hint`: 路径提示可能因节点结构变化
- `diagnostics[].context.*`: 上下文信息因具体操作和环境而异
- `window_handle`: Real 模式下为实际窗口句柄

#### 5.3 诊断字段要求
- `locate_source`: 必须为 `"bounds_center"` 或 `"list_position"`
- `delivery_state`: 必须为 `"sent_unverified"`, `"verified"`, `"failed"` 之一
- `scan_method`: 必须为 `"iaccessible"` (Real) 或 `"mock"` (Mock)
- `focus_method`: 必须为 `"click"`

### 6. Real 模式最小手工验收清单

#### 6.1 Detect 步骤验收
```bash
wechat-debug find-window --json
```
**验收标准：**
- ✅ `status`: `"StatusSuccess"`
- ✅ `instances` 数组非空
- ✅ `confidence` ≥ 0.8
- ✅ `diagnostics` 包含 `window_handle` 信息

#### 6.2 Scan 步骤验收
```bash
wechat-debug scan --json
```
**验收标准：**
- ✅ `status`: `"StatusSuccess"`
- ✅ `conversations` 数组包含至少1个会话
- ✅ 每个会话包含有效的 `display_name` 和 `bounds`
- ✅ `confidence` ≥ 0.8
- ✅ `diagnostics` 包含 `total_conversations` 和 `scan_method`

#### 6.3 Focus 步骤验收
```bash
wechat-debug focus --contact "<联系人>" --json
```
**验收标准：**
- ✅ `status`: `"StatusSuccess"`
- ✅ `confidence` ≥ 0.8
- ✅ `diagnostics` 包含 `locate_source` 和 `click_coordinates`
- ✅ 目标会话在 WeChat 中实际被选中（视觉验证）

#### 6.4 Send 步骤验收
```bash
wechat-debug send --contact "<联系人>" --message "测试消息" --json
```
**验收标准：**
- ✅ `status`: `"StatusSuccess"`
- ✅ `confidence` ≥ 0.8
- ✅ `diagnostics` 包含 `delivery_state: "sent_unverified"`
- ✅ 消息实际出现在 WeChat 输入框并发送（视觉验证）

#### 6.5 Verify 步骤验收
```bash
wechat-debug verify --contact "<联系人>" --message "测试消息" --json
```
**验收标准：**
- ✅ `status`: `"StatusSuccess"` 或合理的失败状态
- ✅ `confidence` ≥ 0.7（验证可能因网络延迟等原因置信度较低）
- ✅ `diagnostics` 包含 `delivery_state` 和 `verification_method`
- ✅ 消息在聊天记录中可见（视觉验证）

### 7. 命令收口说明

#### 7.1 `list-nodes` 命令改名

原 `list-nodes` 命令实际输出的是会话列表，与命令名不符。已统一改名为 `list-conversations`：

```bash
# 旧命令（已废弃）
wechat-debug list-nodes <handle>

# 新命令
wechat-debug list-conversations
```

#### 7.2 真正的节点树查看

如需查看底层的 IAccessible 节点树，请使用 `bridge-dump` 工具：

```bash
# 1. 查找微信窗口句柄
bridge-dump find-wechat

# 2. 查看节点树
bridge-dump list-nodes <handle> --json --depth 3
```

#### 7.3 命令对照表

| 调试目的 | 推荐命令 | 替代命令 |
|----------|----------|----------|
| 查看会话列表 | `wechat-debug list-conversations` | `wechat-debug scan` |
| 查看底层节点树 | `bridge-dump list-nodes <handle>` | N/A |
| 完整链路测试 | `wechat-debug run-chain` | `wechat-debug full-test` |
| 单步骤测试 | `wechat-debug <step>` | N/A |

### 8. 常见问题排查

#### 8.1 Real 模式失败排查步骤

1. **WeChat 未运行**
   ```bash
   # 检查 WeChat 进程
   tasklist | findstr WeChat
   ```

2. **窗口句柄无效**
   ```bash
   # 使用 bridge-dump 验证
   bridge-dump find-wechat
   ```

3. **权限问题**
   - 以管理员身份运行命令行
   - 关闭杀毒软件的实时防护

4. **UI 自动化失败**
   - 确保 WeChat 窗口未被最小化
   - 确保窗口在屏幕可见区域内
   - 检查屏幕缩放设置（建议使用 100%）

#### 8.2 Mock/Real 输出不一致排查

1. **检查诊断字段**
   ```bash
   # Mock 模式
   wechat-debug scan --mock --json | jq '.steps[0].diagnostics'

   # Real 模式
   wechat-debug scan --json | jq '.steps[0].diagnostics'
   ```

2. **验证字段稳定性**
   - 检查必须稳定的字段是否一致
   - 确认允许变化的字段差异合理

3. **回归测试**
   ```bash
   # 运行所有测试
   make test-all
   ```

### 9. 版本兼容性

- **WeChat 版本**: 支持 Windows 桌面版最新版本
- **Windows 版本**: Windows 10/11 (64位)
- **屏幕分辨率**: 建议 1920x1080 或更高
- **DPI 缩放**: 建议 100% (96 DPI)

### 10. 更新日志

- 2026-03-24: 创建 Real 模式联调 SOP 文档
- 2026-03-24: 统一 `list-nodes` 命令改名为 `list-conversations`
- 2026-03-24: 完善 Mock/Real Baseline 输出样例
- 2026-03-24: 补充最小手工验收清单