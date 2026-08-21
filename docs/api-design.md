# FreeDinnerAgent 接口设计草案

## 1. 通用约定

基础路径：

```text
/api/v1
```

请求与响应格式：

```http
Content-Type: application/json
```

通用响应：

```json
{
  "data": {},
  "error": null
}
```

错误响应：

```json
{
  "data": null,
  "error": {
    "code": "BAD_REQUEST",
    "message": "请求参数错误"
  }
}
```

需要登录的接口通过请求头传递访问令牌：

```http
Authorization: Bearer <access_token>
```

## 2. 用户与登录接口

### 注册用户

```http
POST /api/v1/auth/register
```

请求：

```json
{
  "username": "gexinyu",
  "password": "your_password",
  "display_name": "Gexin"
}
```

### 登录

```http
POST /api/v1/auth/login
```

请求：

```json
{
  "username": "gexinyu",
  "password": "your_password"
}
```

响应：

```json
{
  "data": {
    "access_token": "jwt",
    "refresh_token": "opaque_token",
    "user": {
      "id": "uuid",
      "username": "gexinyu",
      "display_name": "Gexin"
    }
  },
  "error": null
}
```

### 当前用户

```http
GET /api/v1/me
```

## 3. Agent 与模型配置接口

### 获取我的 Agent 配置

```http
GET /api/v1/me/agent-config
```

### 更新我的 Agent 配置

```http
PATCH /api/v1/me/agent-config
```

请求：

```json
{
  "name": "默认助理",
  "system_prompt": "你是一个具备长期记忆能力的个人 AI 助理。",
  "default_provider_id": "uuid",
  "temperature": 0.7,
  "max_context_tokens": 12000,
  "max_loop_steps": 6,
  "llm_retry_limit": 2,
  "fallback_policy": {
    "repair_output": true,
    "reduce_context": true,
    "ask_clarification": true,
    "safe_final_answer": true
  },
  "memory_enabled": true,
  "tool_use_enabled": true,
  "dreaming_enabled": true,
  "semantic_memory_enabled": true,
  "embedding_enabled": false,
  "embedding_cost_policy": {
    "mode": "manual",
    "max_monthly_tokens": 0,
    "embed_public_knowledge": false
  }
}
```

### 获取我的模型供应商配置

```http
GET /api/v1/me/model-providers
```

响应中不会返回 API Key 明文，只返回是否已配置。

```json
{
  "data": [
    {
      "id": "uuid",
      "provider": "openai",
      "display_name": "OpenAI 主账号",
      "chat_base_url": "https://api.openai.com/v1",
      "embedding_base_url": "https://api.openai.com/v1",
      "default_chat_model": "gpt-4.1",
      "default_embedding_model": "Pro/BAAI/bge-m3",
      "is_default": true,
      "has_chat_api_key": true,
      "has_embedding_api_key": true,
      "status": "active"
    }
  ],
  "error": null
}
```

### 新增模型供应商配置

```http
POST /api/v1/me/model-providers
```

请求：

```json
{
  "provider": "anthropic",
  "display_name": "Anthropic Claude",
  "chat_base_url": null,
  "chat_api_key": "sk-ant-...",
  "default_chat_model": "claude-3-5-sonnet-latest",
  "embedding_base_url": "https://api.openai.com/v1",
  "embedding_api_key": "sk-...",
  "default_embedding_model": null,
  "is_default": false
}
```

配置说明：`chat_api_key/chat_base_url` 用于对话模型，`embedding_api_key/embedding_base_url` 用于向量化模型。Embedding 未开启时可以不传 embedding 相关字段；后端不会把 API Key 明文返回给前端。

### 更新模型供应商配置

```http
PATCH /api/v1/me/model-providers/{provider_id}
```

### 删除模型供应商配置

```http
DELETE /api/v1/me/model-providers/{provider_id}
```

## 4. 会话接口

### 创建会话

```http
POST /api/v1/conversations
```

请求：

```json
{
  "title": "今天的计划"
}
```

响应：

```json
{
  "data": {
    "id": "uuid",
    "title": "今天的计划",
    "created_at": "2026-08-20T13:00:00Z"
  },
  "error": null
}
```

### 获取会话列表

```http
GET /api/v1/conversations
```

### 获取会话消息

```http
GET /api/v1/conversations/{conversation_id}/messages
```

### 手动压缩当前会话前文

```http
POST /api/v1/conversations/{conversation_id}/compression-jobs
```

请求：

```json
{
  "keep_recent_turns": 8,
  "target_summary_type": "turn_window"
}
```

响应：

```json
{
  "data": {
    "job_id": "uuid",
    "status": "pending"
  },
  "error": null
}
```

该接口用于前端“整理当前对话”按钮。后端会保留最近 N 轮原文，将更早的消息压缩为 `conversation_summaries`，并通过 `conversation_compression_jobs` 记录状态。

### 获取会话压缩任务状态

```http
GET /api/v1/conversations/{conversation_id}/compression-jobs/{job_id}
```

## 5. 对话接口

### 发送消息

```http
POST /api/v1/conversations/{conversation_id}/messages
```

请求：

```json
{
  "content": "我明天下午三点要交报告，帮我记一下。"
}
```

响应：

```json
{
  "data": {
    "turn_id": "uuid",
    "user_message": {
      "id": "uuid",
      "role": "user",
      "content": "我明天下午三点要交报告，帮我记一下。"
    },
    "assistant_message": {
      "id": "uuid",
      "role": "assistant",
      "content": "好的，我已经帮你记录：明天下午三点提交报告。"
    },
    "used_memories": [],
    "tool_calls": [
      {
        "tool_name": "create_task",
        "status": "success"
      }
    ]
  },
  "error": null
}
```

后端会使用当前用户的默认 Agent 配置和默认模型供应商配置生成回复。如果用户没有配置可用模型，接口返回配置缺失错误，引导前端跳转到设置页。

```json
{
  "data": null,
  "error": {
    "code": "MODEL_PROVIDER_REQUIRED",
    "message": "请先在设置中配置 OpenAI 或 Anthropic API Key"
  }
}
```

## 6. 记忆接口

### 获取记忆列表

```http
GET /api/v1/memories?type=preference&status=active
```

### 创建记忆

```http
POST /api/v1/memories
```

请求：

```json
{
  "memory_type": "preference",
  "title": "饮食偏好",
  "content": "用户不喜欢香菜。",
  "importance": 4
}
```

### 更新记忆

```http
PATCH /api/v1/memories/{memory_id}
```

### 删除记忆

```http
DELETE /api/v1/memories/{memory_id}
```

## 7. 任务接口

### 获取任务列表

```http
GET /api/v1/tasks?status=open
```

### 创建任务

```http
POST /api/v1/tasks
```

请求：

```json
{
  "title": "提交报告",
  "description": "下午三点前提交课程报告",
  "due_at": "2026-08-21T15:00:00+08:00"
}
```

### 更新任务状态

```http
PATCH /api/v1/tasks/{task_id}
```

请求：

```json
{
  "status": "done"
}
```

## 8. 心跳任务接口

### 获取我的已安排任务

```http
GET /api/v1/scheduled-agent-jobs?status=active
```

响应：

```json
{
  "data": [
    {
      "id": "uuid",
      "title": "每日简报",
      "job_type": "daily_brief",
      "schedule_kind": "weekly",
      "weekdays": [1, 2, 3, 4, 5],
      "run_at_local_time": "08:00:00",
      "timezone": "Asia/Shanghai",
      "status": "active",
      "next_run_at": "2026-08-24T08:00:00+08:00",
      "last_run_at": null
    }
  ],
  "error": null
}
```

### 获取推荐心跳任务模板

```http
GET /api/v1/scheduled-agent-job-templates
```

第一批模板：

```text
daily_brief
weekly_review
follow_up_monitor
reminder
content_digest
social_assist
```

### 创建心跳任务

```http
POST /api/v1/scheduled-agent-jobs
```

请求：

```json
{
  "title": "每日简报",
  "description": "以日历、未完成任务和重要记忆摘要开启每个工作日。",
  "job_type": "daily_brief",
  "schedule_kind": "weekly",
  "timezone": "Asia/Shanghai",
  "run_at_local_time": "08:00:00",
  "weekdays": [1, 2, 3, 4, 5],
  "prompt_template": "请生成今天的个人简报，重点关注日程、未完成任务和需要我注意的事项。",
  "context_policy": {
    "include_memory": true,
    "include_tasks": true,
    "include_calendar": false,
    "include_email": false,
    "max_context_tokens": 6000
  },
  "tool_policy": {
    "allow_tools": true,
    "requires_approval_for_write": true
  },
  "delivery_channel": "in_app"
}
```

### 更新心跳任务

```http
PATCH /api/v1/scheduled-agent-jobs/{job_id}
```

可更新标题、说明、周期、时间、上下文策略、工具策略和状态。

### 暂停心跳任务

```http
POST /api/v1/scheduled-agent-jobs/{job_id}/pause
```

### 恢复心跳任务

```http
POST /api/v1/scheduled-agent-jobs/{job_id}/resume
```

### 删除心跳任务

```http
DELETE /api/v1/scheduled-agent-jobs/{job_id}
```

逻辑删除，更新 `status = deleted`。

### 立即运行心跳任务

```http
POST /api/v1/scheduled-agent-jobs/{job_id}/run-now
```

响应：

```json
{
  "data": {
    "run_id": "uuid",
    "status": "pending"
  },
  "error": null
}
```

### 获取心跳任务运行记录

```http
GET /api/v1/scheduled-agent-jobs/{job_id}/runs
```

### 获取单次运行详情

```http
GET /api/v1/scheduled-agent-job-runs/{run_id}
```

返回本次触发时间、执行状态、关联会话、关联 Agent Turn、输出摘要和失败原因。

## 9. Agent 执行与工具调用日志接口

### 获取会话 Agent 执行事件

```http
GET /api/v1/conversations/{conversation_id}/agent-events?turn_id=uuid
```

该接口用于前端展示一次 Agent 执行过程，例如上下文构建、loop step、工具路由、工具调用、审批请求、兜底触发和最终回复。

### 获取 Agent Turn 详情

```http
GET /api/v1/conversations/{conversation_id}/agent-turns/{turn_id}
```

返回一次用户输入到助手回复的完整执行状态，包括模型供应商、Agent 配置、状态、错误信息、开始时间和结束时间。

### 获取 Agent Loop 步骤

```http
GET /api/v1/conversations/{conversation_id}/agent-turns/{turn_id}/loop-steps
```

返回 Bounded ReAct 的步骤记录：

```json
{
  "data": [
    {
      "step_no": 1,
      "step_type": "reason",
      "thought_summary": "需要把用户请求转成一个待办任务。",
      "action_type": "tool_call",
      "status": "success"
    },
    {
      "step_no": 2,
      "step_type": "observe",
      "observation": "create_task 执行成功，任务 id 为 uuid。",
      "status": "success"
    }
  ],
  "error": null
}
```

`thought_summary` 只保存可审计摘要，不返回隐藏思维链。

### 获取 LLM 输出校验记录

```http
GET /api/v1/conversations/{conversation_id}/agent-turns/{turn_id}/validations
```

该接口用于调试 JSON Schema 校验、工具参数校验、安全策略校验、最终回复声明校验和记忆写入策略校验。

### 获取 Agent 兜底记录

```http
GET /api/v1/conversations/{conversation_id}/agent-turns/{turn_id}/fallback-events
```

该接口用于查看本轮是否触发过输出修复、LLM 重试、上下文压缩、工具禁用、澄清提问或保守兜底回复。

### 获取指定会话的工具调用记录

```http
GET /api/v1/conversations/{conversation_id}/tool-calls
```

该接口用于调试和展示 Agent 的执行过程，前端可以在对话详情中显示工具调用状态。

## 10. 工具配置接口

### 获取可用工具列表

```http
GET /api/v1/tools
```

响应：

```json
{
  "data": [
    {
      "id": "uuid",
      "name": "create_task",
      "display_name": "创建任务",
      "category": "task",
      "permission_level": "normal",
      "requires_approval": false,
      "is_enabled": true
    }
  ],
  "error": null
}
```

### 获取我的工具设置

```http
GET /api/v1/me/tool-settings
```

### 更新我的工具设置

```http
PATCH /api/v1/me/tool-settings/{tool_id}
```

请求：

```json
{
  "is_enabled": true,
  "approval_policy": "sensitive_only"
}
```

### 获取某次工具调用详情

```http
GET /api/v1/tool-calls/{tool_call_id}
```

该接口用于展示工具执行过程、参数校验结果、重试次数、耗时和失败原因。

### 审批工具调用

```http
POST /api/v1/tool-approval-requests/{approval_id}/approve
```

### 拒绝工具调用

```http
POST /api/v1/tool-approval-requests/{approval_id}/reject
```

工具涉及敏感或破坏性操作时，后端会先创建审批请求，并将 Agent Turn 标记为 `waiting_approval`。用户审批后，后端继续执行或取消该工具调用。

## 11. 多渠道入口接口

### 获取可用 Channel Provider

```http
GET /api/v1/channel-providers
```

响应：

```json
{
  "data": [
    {
      "id": "uuid",
      "name": "napcatqq",
      "display_name": "NapCatQQ",
      "provider_type": "qq",
      "adapter_type": "http_webhook",
      "inbound_modes": ["http_webhook", "websocket"],
      "outbound_modes": ["send_message"]
    }
  ],
  "error": null
}
```

### 创建渠道连接

```http
POST /api/v1/me/channel-connections
```

请求：

```json
{
  "provider_id": "uuid",
  "display_name": "我的 NapCatQQ",
  "external_account_id": "123456",
  "external_account_name": "FreeDinnerBot",
  "config": {
    "endpoint": "http://127.0.0.1:3000",
    "access_token": "secret",
    "webhook_secret": "secret"
  }
}
```

`config` 中的敏感字段加密写入 `channel_connections.encrypted_config`。

### 获取我的渠道连接

```http
GET /api/v1/me/channel-connections
```

### 更新渠道连接状态

```http
PATCH /api/v1/me/channel-connections/{connection_id}
```

可用于暂停、恢复、删除连接，或更新健康检查配置。

### 配置渠道策略

```http
PATCH /api/v1/me/channel-connections/{connection_id}/policies
```

请求：

```json
{
  "scope_type": "group_chat",
  "external_scope_id": "987654",
  "mode": "mention_only",
  "trigger_keywords": ["小饭", "agent"],
  "allow_memory_write": true,
  "allow_tool_use": true,
  "require_approval_for_outbound": true,
  "rate_limit_per_minute": 6
}
```

### Channel Webhook 入口

```http
POST /api/v1/channels/{connection_id}/webhook
```

该接口由 NapCatQQ、飞书、Telegram webhook 等外部平台调用。后端需要校验签名或 secret，写入 `channel_inbox_events`，再按 `channel_policies` 判断是否触发 Agent。

### 获取外部会话映射

```http
GET /api/v1/me/channel-connections/{connection_id}/external-conversations
```

### 获取渠道入站事件

```http
GET /api/v1/me/channel-connections/{connection_id}/inbox-events
```

### 获取渠道待发送消息

```http
GET /api/v1/me/channel-connections/{connection_id}/outbox-messages?status=pending
```

### 审批渠道外发消息

```http
POST /api/v1/channel-outbox-messages/{outbox_id}/approve
```

### 取消渠道外发消息

```http
POST /api/v1/channel-outbox-messages/{outbox_id}/cancel
```

群聊消息、社交辅助消息和高风险外发默认进入 outbox 等待审批或草稿确认。

## 12. 能力市场接口

### 获取能力市场列表

```http
GET /api/v1/marketplace?type=skill&visibility=public
```

能力类型包括：

```text
tool
mcp_server
skill
knowledge_base
channel_adapter
```

### 安装能力

```http
POST /api/v1/marketplace/{item_id}/install
```

响应：

```json
{
  "data": {
    "install_id": "uuid",
    "capability_type": "skill",
    "capability_ref_id": "uuid",
    "is_enabled": true
  },
  "error": null
}
```

### 卸载能力

```http
DELETE /api/v1/me/capability-installs/{install_id}
```

### 获取我的已安装能力

```http
GET /api/v1/me/capability-installs
```

### 将能力启用到我的 Agent

```http
POST /api/v1/me/agent-config/capabilities
```

请求：

```json
{
  "capability_type": "skill",
  "capability_ref_id": "uuid",
  "load_mode": "auto",
  "priority": 10
}
```

### 更新 Agent 能力绑定

```http
PATCH /api/v1/me/agent-config/capabilities/{binding_id}
```

请求：

```json
{
  "is_enabled": true,
  "load_mode": "standard",
  "priority": 20
}
```

### 配置 MCP Server

```http
PATCH /api/v1/me/mcp-servers/{mcp_server_id}/settings
```

请求：

```json
{
  "is_enabled": true,
  "approval_policy": "sensitive_only",
  "env": {
    "API_TOKEN": "secret"
  }
}
```

`env` 中的敏感信息由后端加密保存到 `user_mcp_server_settings.encrypted_env`，接口不返回明文。

## 13. 系统提示词市场接口

### 获取系统提示词模板市场

```http
GET /api/v1/system-prompt-templates?visibility=public&category=assistant
```

返回公共模板和当前用户自己的私有模板。

### 创建系统提示词模板

```http
POST /api/v1/system-prompt-templates
```

请求：

```json
{
  "display_name": "学习型个人助理",
  "description": "适合课程复习、资料整理和项目推进的系统提示词。",
  "category": "study",
  "tags": ["study", "planner"],
  "visibility": "private"
}
```

### 创建模板版本

```http
POST /api/v1/system-prompt-templates/{template_id}/versions
```

请求：

```json
{
  "content": "你是 {user_display_name} 的学习型个人助理，回答时先给结论，再给步骤。",
  "change_note": "初始版本",
  "variables": [
    {
      "name": "user_display_name",
      "value_type": "string",
      "required": true,
      "default_value": "用户"
    }
  ]
}
```

### 绑定模板到我的 Agent

```http
PATCH /api/v1/me/agent-config/system-prompt
```

请求：

```json
{
  "system_prompt_template_id": "uuid",
  "system_prompt_template_version_id": "uuid",
  "system_prompt_variables": {
    "user_display_name": "Gexin"
  },
  "custom_system_prompt_override": "回答尽量简洁，但保留关键推理步骤。"
}
```

Agent 绑定的是具体模板版本。公共模板发布新版本后，不会自动改变用户 Agent 行为。

### 预览最终系统提示词

```http
POST /api/v1/me/agent-config/system-prompt/preview
```

该接口返回模板、变量、用户覆盖内容和系统策略拼接后的预览，用于前端确认。

## 14. Workspace Sandbox 接口

### 获取我的 Workspace 状态

```http
GET /api/v1/me/workspace
```

响应包含 workspace 是否启用、状态、磁盘配额、文件数量、网络策略和最近活跃时间。

### 启用 Workspace

```http
POST /api/v1/me/workspace
```

请求：

```json
{
  "sandbox_type": "local_dir",
  "max_disk_bytes": 1073741824,
  "max_file_count": 5000,
  "max_single_file_bytes": 52428800,
  "max_command_seconds": 30,
  "network_policy": "disabled",
  "cpu_limit": "1.0",
  "memory_limit_bytes": 536870912,
  "idle_after_seconds": 604800,
  "destroy_after_seconds": 2592000
}
```

### 更新 Workspace 策略

```http
PATCH /api/v1/me/workspace
```

请求：

```json
{
  "network_policy": "allowlist",
  "network_allowlist": ["github.com", "registry.npmjs.org"],
  "max_command_seconds": 60
}
```

### 列出文件

```http
GET /api/v1/me/workspace/files?path=/
```

### 读取文件

```http
GET /api/v1/me/workspace/files/content?path=/notes/todo.md
```

### 写入文件

```http
PUT /api/v1/me/workspace/files/content
```

请求：

```json
{
  "path": "/notes/todo.md",
  "content": "今天要完成 Step 11 设计。"
}
```

### 执行受限命令

```http
POST /api/v1/me/workspace/commands
```

请求：

```json
{
  "command": "go",
  "args": ["test", "./..."],
  "working_dir": "/project",
  "timeout_seconds": 30
}
```

命令执行必须限制在当前用户 workspace 内，默认禁止网络。生产环境必须使用容器或更强 sandbox。

当前后端默认使用本地目录 sandbox 和白名单命令执行，不经过 shell。参数禁止绝对路径、`..`、换行和 NUL 字符；`python`、`python3`、`node` 只允许运行 workspace 内脚本文件。

如果用户 workspace 的 `sandbox_type` 是 `docker`、`podman` 或 `nsjail`，后端会将同一条命令转交给对应 runner，并使用 `network_policy`、`cpu_limit`、`memory_limit_bytes`、`max_command_seconds` 生成隔离参数。本地开发不要求安装这些 runtime；部署环境再安装并配置对应 binary。

### 查看命令记录

```http
GET /api/v1/me/workspace/commands?limit=50
```

### 销毁 Workspace

```http
DELETE /api/v1/me/workspace
```

销毁前可以选择归档或直接删除。MVP 阶段建议只支持用户手动销毁。
