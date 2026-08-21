# FreeDinnerAgent 工具调用设计

## 1. 设计目标

工具调用模块负责把 LLM 的自然语言意图转换成稳定、可审计、可扩展的函数调用。设计重点不是“能调一个函数”，而是让工具具备注册、路由、参数校验、权限控制、重试、降级和观测能力。

参考 OpenAI Agents SDK 和 Codex as a platform 的设计思路，工具系统不只是函数列表，而是由宿主应用提供上下文、工具、审批、事件流和持久化记录。FreeDinnerAgent 在 Go 后端中实现自己的 Tool Registry、Tool Router、Tool Executor 和 Agent Harness，并由 Bounded ReAct loop 驱动工具选择、执行和观察。

## 2. 总体流程

```text
用户输入
  ↓
创建 Agent Turn
  ↓
构建上下文并进入 Bounded ReAct Loop
  ↓
Tool Router 判断是否需要工具
  ↓
从 Tool Registry 获取候选工具
  ↓
按用户配置、权限、任务类型过滤
  ↓
把少量候选工具 schema 注入 Prompt
  ↓
LLM 生成 tool call
  ↓
Tool Executor 校验参数
  ↓
执行工具、超时控制、重试
  ↓
结果结构化压缩
  ↓
交还 LLM 生成最终回复
  ↓
写入 tool_call_logs / episode_tool_calls
  ↓
写入 agent_events，前端展示执行过程
```

Agent loop、LLM 输出校验、修复、重试和降级的完整设计见 [Agent Loop 与可靠性设计](agent-loop-reliability-design.md)。

## 3. Agent Harness

Agent Harness 是一次 Agent 执行的外壳，负责管理生命周期、事件流、取消、审批和持久化。

存储表：

```text
agent_turns
agent_events
```

`agent_turns` 表示一次用户输入到助手最终回复的完整执行过程，状态包括：

- `pending`
- `running`
- `waiting_approval`
- `success`
- `failed`
- `cancelled`

`agent_events` 是事件流，前端可以用它展示执行过程：

- `turn_started`
- `context_built`
- `loop_step_started`
- `loop_step_finished`
- `tool_routed`
- `tool_call_started`
- `tool_call_finished`
- `approval_requested`
- `approval_resolved`
- `llm_validation_failed`
- `fallback_triggered`
- `message_delta`
- `message_completed`
- `turn_failed`
- `turn_cancelled`

这样前端不只是看到最终答案，还能看到“正在检索记忆”“准备调用工具”“等待用户确认”“工具执行完成”等过程。

## 4. 工具注册

工具元数据写入：

```text
tool_definitions
tool_versions
```

`tool_definitions` 存稳定信息：

- `name`：工具唯一名，例如 `create_task`。
- `namespace`：工具命名空间，例如 `memory`、`task`。
- `description`：给模型看的工具说明。
- `category`：工具分类，用于路由。
- `handler_type`：`builtin`、`http`、`mcp`、`agent`。
- `handler_ref`：后端执行器引用，例如 Go 函数名、HTTP URL、MCP tool name。
- `permission_level`：`readonly`、`normal`、`sensitive`、`destructive`。
- `requires_approval`：是否需要用户确认。
- `timeout_ms`、`max_retries`、`retry_backoff_ms`：稳定性策略。
- `is_enabled`：系统级启用开关。

`tool_versions` 存 schema：

- `parameter_schema`：JSON Schema，用于 LLM tool definition 和后端参数校验。
- `result_schema`：工具结果结构。
- `version`：schema 版本。
- `status`：active、deprecated、deleted。

这样新增或修改工具时，可以保留旧版本，避免历史工具调用日志无法解释。

## 5. 用户级工具配置

用户可通过：

```text
user_tool_settings
```

控制某个工具是否启用、是否需要审批、以及工具私有配置。

典型场景：

- 用户关闭信息检索工具。
- 用户要求所有敏感工具调用都必须确认。
- 某些工具需要用户自己的外部服务配置。

工具私有配置写入 `encrypted_config`，不保存明文密钥。

## 6. 工具路由

Tool Router 的目标是减少 Prompt 中的工具数量，避免所有工具 schema 都塞给模型。

路由输入：

- 当前用户输入
- 当前会话 Working Memory
- Agent 配置
- 用户启用的工具列表
- 命中的 Procedural Skill
- 当前任务类型

路由规则：

- 简单闲聊：不加载工具。
- 明确记忆请求：加载 `save_profile_memory`、`search_memory`。
- 任务管理：加载 `create_task`、`list_tasks`、`update_task`。
- 心跳任务：加载 `create_scheduled_agent_job`、`list_scheduled_agent_jobs`、`pause_scheduled_agent_job`。
- 社交渠道：加载 `send_channel_message`、`list_channel_messages`、`get_channel_members`，高风险外发需要审批。
- 文档相关：加载 `ingest_document`、`search_semantic_memory`。
- 命中技能：加载该 skill 声明的工具序列。

路由结果写入：

```text
tool_router_logs
```

用于排查为什么某个工具被选中或没被选中。

## 7. 参数校验

LLM 生成 tool call 后，后端不能直接执行。

校验步骤：

1. 根据 `tool_name` 找到 active `tool_definitions`。
2. 找到最新 active `tool_versions`。
3. 使用 `parameter_schema` 校验参数。
4. 检查用户是否启用该工具。
5. 检查权限级别和审批策略。
6. 生成 `idempotency_key`，防止重复执行。
7. 参数通过后再进入执行器。

校验失败时：

- 写入 `tool_call_logs`。
- 不执行工具。
- 返回可理解的错误给 LLM，让它修正参数或直接回复用户。

## 8. 工具稳定性处理

本章节只处理工具执行层面的稳定性。LLM 输出格式错误、模型调用失败、上下文超限和最终回复不可靠等问题，由 Agent Validator 和 Fallback Manager 处理。

### 8.1 超时

每个工具有自己的 `timeout_ms`。超过时间后：

- 标记 `status = timeout`。
- 记录 `error_type = timeout`。
- 如果工具可降级，返回 fallback 结果。

### 8.2 重试

只对可重试错误重试：

- 网络抖动
- 临时 5xx
- 数据库短暂连接失败

不重试：

- 参数校验失败
- 权限不足
- 用户拒绝审批
- 明确业务失败

重试次数由 `max_retries` 控制，退避时间由 `retry_backoff_ms` 控制。

### 8.3 幂等

会改变状态的工具必须支持 `idempotency_key`。

例如：

- 创建任务
- 保存记忆
- 导入文档

如果同一个 `idempotency_key` 已成功执行，后端直接返回已有结果，避免重复创建。

### 8.4 降级

工具失败后不应该让对话中断。

降级方式：

- 检索失败：改用已有上下文回答，并说明没有查到新资料。
- 创建任务失败：返回待确认草稿，让用户稍后重试。
- 文档导入失败：保留原文，后台异步重试。
- 模型工具调用格式错误：写入 `llm_output_validations`，优先让 Validator 修复结构化输出，修复失败后再按 Agent 配置重试。

## 9. 审批机制

需要审批的工具：

- `permission_level = sensitive`
- `permission_level = destructive`
- `requires_approval = true`
- 用户设置 `approval_policy = always`

审批前，后端返回待确认状态给前端：

```text
工具：create_task
动作：创建待办事项
参数：明天下午三点提交报告
风险：normal
```

用户确认后再执行。拒绝则写入 `tool_call_logs.status = cancelled`。

审批请求写入：

```text
tool_approval_requests
```

审批过程：

1. Tool Executor 发现工具需要审批。
2. 创建 `tool_approval_requests`。
3. `agent_turns.status` 更新为 `waiting_approval`。
4. 写入 `agent_events.event_type = approval_requested`。
5. 前端展示确认面板。
6. 用户批准或拒绝。
7. 写入 `approval_resolved` 事件。
8. 批准则继续执行工具，拒绝则取消该工具调用。

## 10. 调用日志

工具运行时写入：

```text
tool_call_logs
```

记录内容：

- 工具名称和版本
- 原始参数和校验后参数
- 结果
- 状态
- 错误类型和错误信息
- 尝试次数
- 耗时
- 是否需要审批

对话复盘时，重要工具调用还会进入 `episode_tool_calls`，成为 Episodic Memory 的一部分。

## 11. 新增工具扩展流程

新增工具遵循固定步骤：

1. 实现后端 handler。
2. 定义工具 `parameter_schema` 和 `result_schema`。
3. 向 `tool_definitions` 插入工具元数据。
4. 向 `tool_versions` 插入 schema 版本。
5. 在 Tool Router 中增加路由规则或关键词。
6. 如涉及用户配置，增加 `user_tool_settings` 默认配置。
7. 编写参数校验测试、成功执行测试、失败降级测试。
8. 在前端设置页展示该工具开关。

示例：

```sql
INSERT INTO tool_definitions (
    id,
    name,
    namespace,
    display_name,
    description,
    category,
    handler_type,
    handler_ref,
    permission_level,
    requires_approval
) VALUES (
    gen_random_uuid(),
    'search_semantic_memory',
    'memory',
    '检索知识库',
    '从用户私有知识库和公共知识库中检索相关文档片段。',
    'memory',
    'builtin',
    'memory.SearchSemanticMemory',
    'readonly',
    false
);
```

## 12. 第一批工具

MVP 阶段建议先实现：

- `save_profile_memory`：保存用户画像记忆。
- `search_memory`：检索多层记忆。
- `create_task`：创建任务。
- `list_tasks`：查询任务。
- `update_task`：更新任务状态。
- `create_scheduled_agent_job`：创建每日简报、每周回顾、跟进监控等心跳任务。
- `list_scheduled_agent_jobs`：查询心跳任务。
- `run_scheduled_agent_job`：立即运行一次心跳任务。
- `send_channel_message`：通过 QQ、Telegram、Discord、飞书等渠道发送消息。
- `list_channel_messages`：读取用户授权渠道中的最近消息。
- `get_channel_members`：查询群聊或频道成员。
- `ingest_document`：导入文档到 Semantic Memory。
- `search_semantic_memory`：检索知识库。
- `summarize_text`：总结长文本。

## 13. 小巧思：工具健康评分

Channel Adapter 负责“外部消息进入系统”，工具负责“Agent 主动执行平台动作”。例如 NapCatQQ 的完整接入可以同时包含：

```text
NapCatQQ Channel Adapter
  -> 接收 QQ 私聊和群聊事件
  -> 触发 Agent Turn

NapCatQQ Tools / MCP
  -> send_private_message
  -> send_group_message
  -> list_group_members
  -> get_recent_messages
```

这样 QQ、Telegram、Discord、飞书都能复用同一套入口与工具边界。

系统可以根据 `tool_call_logs` 计算工具健康评分：

```text
tool_health = success_rate - timeout_rate - fallback_rate
```

Tool Router 优先选择健康评分高的工具。如果某个工具近期连续失败，就自动降低路由优先级，避免 Agent 反复调用坏工具。

## 14. 小巧思：工具调用回放

前端可以提供“查看执行过程”：

```text
1. 路由选中 create_task
2. 参数校验通过
3. 创建任务成功
4. 写入任务表
5. 助手生成最终回复
```

这能提升可解释性，也方便调试 Function Calling 的稳定性。

回放数据来自：

```text
agent_events
tool_router_logs
tool_call_logs
tool_approval_requests
context_build_logs
```

## 15. 参考资料

- [OpenAI Developers: Codex as a platform](https://developers.openai.com/blog/codex-as-a-platform)
- [OpenAI API: Model guidance](https://developers.openai.com/api/docs/guides/latest-model)
- [OpenAI Agents SDK: Tools](https://openai.github.io/openai-agents-python/tools/)
