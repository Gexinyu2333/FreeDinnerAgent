# FreeDinnerAgent 多渠道入口设计

## 1. 设计目标

FreeDinnerAgent 的 Web 页面是主控制台，但真实使用场景不应该只停留在网页里。用户可以在 QQ、微信、Telegram、Discord、飞书等聊天入口直接和自己的 Agent 对话，也可以让 Agent 在授权群聊里监听、总结或按规则回复。

第一阶段建议先接入 NapCatQQ，但架构上不要把 QQ 写死成特例，而是设计成通用 Channel Adapter：

```text
QQ / Telegram / Discord / Feishu / WeChat
  ↓
Channel Adapter
  ↓
FreeDinnerAgent Channel API
  ↓
Conversation / Message / Agent Turn
  ↓
Agent Harness / Memory / Tool / MCP
  ↓
Channel Outbox
  ↓
外部平台发回消息
```

## 2. Channel 和 Tool/MCP 的区别

这点很关键：

- Channel Adapter：负责接收外部消息，把外部聊天变成本地 conversation。
- Tool/MCP：负责 Agent 主动调用外部能力，例如发 QQ 消息、查群成员、取最近消息。

以 NapCatQQ 为例：

```text
收到 QQ 消息 -> Channel Adapter
发送 QQ 消息 -> Tool 或 MCP
```

如果只把 NapCatQQ 包成 MCP，Agent 可以发消息，但“QQ 消息主动触发 Agent”这条入口链路会不完整。所以完整设计应该是：

```text
NapCatQQ Channel Adapter + NapCatQQ MCP Tools
```

## 3. 通用抽象

### 3.1 Channel Provider

`channel_provider_definitions` 描述一种平台或适配器：

- `qq`：NapCatQQ / OneBot。
- `wechat`：微信个人号或企业微信适配器。
- `telegram`：Telegram Bot。
- `discord`：Discord Bot。
- `feishu`：飞书机器人。
- `custom`：自定义 webhook。

Provider 保存平台级能力：

- 支持哪些入站模式：webhook、websocket、polling。
- 支持哪些出站模式：send_message、send_image、send_card。
- 配置 schema。
- 默认触发策略。

### 3.2 Channel Connection

`channel_connections` 描述某个用户绑定的具体实例。

例如：

```text
用户 A 的 NapCatQQ 实例
用户 A 的 Telegram Bot
用户 B 的飞书机器人
```

敏感配置写入 `encrypted_config`，例如 token、webhook secret、NapCat endpoint。

### 3.3 External Conversation

`external_conversations` 映射外部聊天和本地会话：

```text
QQ 好友 12345 -> conversation_id
QQ群 67890 -> conversation_id
Telegram chat 111 -> conversation_id
Discord channel 222 -> conversation_id
```

这样 Agent 的内部逻辑永远只面对统一的 conversation，不需要知道消息来自 QQ 还是飞书。

### 3.4 Inbox / Outbox

`channel_inbox_events` 保存外部平台进来的原始事件和规范化文本。

`channel_outbox_messages` 保存 Agent 准备发出的消息、审批状态、发送状态和外部消息 ID。

这样做有几个好处：

- 可以去重。
- 可以重试。
- 可以审计。
- 可以限流。
- 可以在失败时保留草稿。

## 4. QQ 接入方式

NapCatQQ 适合作为第一条落地通道。

推荐流程：

```text
NapCatQQ / OneBot 收到私聊或群聊事件
  ↓
POST /api/v1/channels/{connection_id}/webhook
  ↓
校验 secret
  ↓
写入 channel_inbox_events
  ↓
按 channel_policies 判断是否触发 Agent
  ↓
查找或创建 external_conversations
  ↓
写入 messages
  ↓
创建 agent_turns
  ↓
Agent 回复写入 channel_outbox_messages
  ↓
NapCatQQ send_msg API 发送
```

如果使用 WebSocket，则由 `internal/channel/qq` 维持连接，收到事件后走同一套内部处理流程。

## 5. 群聊策略

群聊比私聊更容易失控，必须有策略。

建议模式：

- `disabled`：不监听。
- `silent_listen`：只记录和总结，不主动回复。
- `mention_only`：只有 @Agent 时回复。
- `keyword`：命中特定关键词时回复。
- `auto_reply`：自动回复，但需要严格限频。

默认策略：

```text
私聊：允许自动回复
群聊：默认 mention_only
社交发送：默认需要审批
群聊自动发言：默认关闭
```

这样既能体现社交辅助能力，又不会让 Agent 在群里乱说话。

## 6. 多平台扩展

新增一个平台时，不应该重写 Agent，只需要实现一个 Adapter。

统一接口：

```go
type ChannelAdapter interface {
    VerifyInbound(ctx context.Context, req InboundRequest) error
    NormalizeEvent(ctx context.Context, raw []byte) (*ChannelEvent, error)
    SendMessage(ctx context.Context, msg OutboxMessage) (*SendResult, error)
    HealthCheck(ctx context.Context, conn ChannelConnection) (*HealthStatus, error)
}
```

各平台只处理差异：

```text
NapCatQQAdapter      # OneBot event / send_msg
TelegramAdapter     # Telegram Bot API
DiscordAdapter      # Discord Gateway / REST
FeishuAdapter       # 飞书事件订阅 / 消息 API
WeChatAdapter       # 微信或企业微信事件
```

Agent 层不关心平台差异，只接收统一后的消息。

## 7. 和能力市场的关系

Channel Adapter 可以进入能力市场，类型为：

```text
channel_adapter
```

市场里展示：

- NapCatQQ Channel
- Telegram Bot Channel
- Discord Bot Channel
- 飞书机器人 Channel
- 自定义 Webhook Channel

用户安装后，还需要完成连接配置：

```text
安装 Channel Adapter
  ↓
创建 channel_connection
  ↓
配置密钥、endpoint、secret
  ↓
选择启用到哪个 Agent
  ↓
配置私聊/群聊策略
```

## 8. 和 MCP 的关系

Channel Adapter 负责入口，MCP 负责能力。

一个平台可以同时提供两类能力：

```text
NapCatQQ Channel Adapter
  - 收 QQ 私聊和群聊
  - 建本地 conversation
  - 触发 Agent

NapCatQQ MCP Server
  - send_private_message
  - send_group_message
  - list_group_members
  - get_recent_messages
```

这两个可以共享同一个 `channel_connection` 的配置，也可以分别配置。MVP 阶段建议先让 Channel Adapter 直接调用 NapCatQQ 发消息，后续再把更多 QQ 操作抽成 MCP tools。

## 9. 安全与隐私

多渠道入口会接触大量真实聊天数据，必须默认保守。

规则：

- 所有 channel 表都保存 `user_id`。
- 外部事件先落 `channel_inbox_events`，再决定是否触发 Agent。
- 群聊默认只在 @Agent 时触发。
- 发送群消息默认需要审批或至少限频。
- 社交辅助类自动发送默认关闭，先生成草稿。
- 用户可以给每个群配置不同 Agent、关键词和静默时段。
- 日志中避免保存敏感附件原文，附件只保存引用和摘要。
- 用户可以停用或删除 channel connection。

## 10. MVP 建议

第一版只做 NapCatQQ：

1. `channel_provider_definitions` 内置 NapCatQQ。
2. Web 设置页允许用户配置 NapCat endpoint、access token、secret。
3. 支持 QQ 私聊触发 Agent。
4. 支持 QQ 群聊 @Agent 触发。
5. 回复先走文本消息。
6. 群聊发送限频。
7. 所有入站和出站消息有日志。

第二阶段再做：

- 群聊静默总结。
- 图片和文件消息摘要。
- QQ MCP tools。
- Telegram Bot。
- 飞书机器人。
- Discord Bot。

这样既有明确 MVP，又能自然扩展到更多入口。
