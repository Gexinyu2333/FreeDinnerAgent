# Backend

本目录用于存放 FreeDinnerAgent 的 Go 后端服务。

计划技术栈：

- Go
- Gin
- PostgreSQL Driver
- LLM API Client

核心模块规划：

- `cmd/server`：服务启动入口
- `internal/app`：组合根，负责依赖组装、内置能力同步和后台 worker 启动
- `internal/api`：HTTP handler、middleware、统一响应和路由注册
- `internal/agent`：Agent 编排、上下文管理、工具调用决策
- `internal/channel`：Channel Adapter 抽象、NapCatQQ / OneBot webhook、outbox 和 sender worker
- `internal/contextmgr`：上下文拼装、token 阈值、摘要压缩和上下文体检
- `internal/knowledge`：知识库文档切片、embedding 和检索
- `internal/llm`：OpenAI-compatible LLM 客户端、聊天服务和 Agent Loop 入口
- `internal/market`：Tools、MCP、Skills、知识库和系统提示词市场
- `internal/mcp`：MCP definition runtime 骨架、metadata discovery 和 tool sync
- `internal/memory`：记忆写入、检索和管理
- `internal/scheduler`：心跳任务调度、到期扫描和运行记录
- `internal/secret`：用户 API Key、渠道 token 等敏感配置加解密
- `internal/tool`：工具注册、参数校验、执行和降级处理
- `internal/workspace`：用户 workspace、文件操作、CLI sandbox 和资源配额
- `internal/store`：数据库访问层
- `internal/config`：环境变量和配置加载

当前约定：`internal/app/app.go` 串主依赖图，`stores.go` 创建 store，`bootstrap.go` 做启动期同步；`internal/api/router.go` 只创建 Gin router，具体路由按 `routes_*.go` 拆分。

`internal/api` 的 handler 保持 HTTP 边界职责：解析 path/query/body、读取当前用户、做轻量请求校验、调用 service/store facade，并使用统一 `OK/Error` 返回。`settings_handler.go`、`scheduled_job_handler.go`、`market_handler.go`、`workspace_handler.go` 当前仍按入口域聚合 DTO 和 handler 方法；由于其主要复杂度来自请求结构和错误映射，暂不为行数继续拆散。

`internal/channel` 已按职责拆分：`service.go` 保留连接和策略入口，`webhook.go` 负责入站事件与 Agent 回复草稿，`outbox.go` 负责审批和发送，`policy.go` 负责触发和限频，`onebot.go` 负责 OneBot 协议细节，`config_crypto.go` 负责连接配置加解密。

`internal/workspace` 也按职责拆分：`service.go` 管生命周期和状态，`files.go` 管文件 I/O，`command.go` 管 CLI 执行和运行记录，`path.go` 管路径作用域和逃逸防护，`policy.go` 管默认策略、命令白名单和参数安全检查，`runner.go` 管 local/container/nsjail 执行器。

`internal/scheduler` 按心跳任务职责拆分：`service.go` 管任务 CRUD，`worker.go` 管后台扫描，`execution.go` 管触发执行和运行记录，`schedule.go` 管下次运行时间与 cron 解析，`templates.go` 管内置建议模板，`update.go` 管局部更新 merge。

`internal/tool` 按工具调用职责拆分：`service.go` 管主执行流水线，`routing.go` 管 Tool Router 和 Agent 绑定过滤，`approval.go` 管审批策略和 dry-run，`builtin_definitions.go` 管内置工具定义，`builtins.go` 管内置工具实现，`mcp.go` 管 MCP HTTP bridge，`schema.go` 管参数校验。

`internal/memory` 按记忆层职责拆分：`manager.go` 管接口和核心类型，`lifecycle.go` 管 working/episode 写入，`retrieval.go` 管统一检索和压缩，`chunks.go` 管层到 chunk 的转换，`skills.go` 管 Procedural Skill，`dreaming.go` 管 dreaming insight，`curator.go` 管 curator job，`util.go` 管 token 估算和结果统计。

`internal/llm` 按主链路职责拆分：`service.go` 管 Web/Channel 发送入口，`agent_loop.go` 管 ReAct 主循环，`loop_observation.go` 管 memory/tool observation，`loop_finish.go` 管 turn 收尾，`loop_recording.go` 管校验和 fallback 记录，`curation.go` 管对话后记忆沉淀，`context.go` 管上下文构建，`features.go` 管额外 LLM feature provider，`compression.go` 管自动压缩，`openai.go` 管兼容 OpenAI 的 HTTP client。

`internal/market` 按服务职责拆分：`service.go` 保留 marketplace 列表、安装、评分、绑定和系统提示词模板主用例，`prompt_template.go` 管系统提示词变量提取、渲染、值校验、token 估算和安全扫描。

`internal/contextmgr` 按上下文构建阶段拆分：`builder.go` 保留 Build 主流程、summary 加载和 build log 写入，`budget.go` 管 token 预算，`messages.go` 管最近消息选择和规则摘要，`render.go` 管 memory/skill/summary 渲染，`items.go` 管 context build item 记录，`tokens.go` 管 token 估算、压缩策略和 used sections。

`internal/store` 按数据库领域拆分：`memory_store.go` 只保留记忆相关类型和构造器，`memory_profile_store.go` 管 Profile Memory 和类型定义查询，`memory_working_store.go` 管 Working Memory，`memory_episode_store.go` 管 Episodic Memory，`memory_skill_store.go` 管 Procedural Skill，`memory_dreaming_store.go` 管 Dreaming，`memory_curator_store.go` 管 Curator Job，`memory_retrieval_log_store.go` 管检索日志，`memory_scan.go` 管 scan helper。

`internal/store` 的 Channel 部分也按表域拆分：`channel_store.go` 只保留 Channel 类型、构造器和 public DTO 转换，`channel_provider_store.go` 管 provider definition，`channel_connection_store.go` 管用户连接，`channel_policy_store.go` 管触发策略，`channel_conversation_store.go` 管外部会话映射，`channel_inbox_store.go` 管入站事件和限频计数，`channel_outbox_store.go` 管外发消息状态流转，`channel_scan.go` 管 scan helper。

`internal/store` 的 Market 部分按能力市场子域拆分：`market_store.go` 只保留 marketplace、install、binding 和 system prompt 类型，`marketplace_item_store.go` 管市场条目、上架和评分，`market_capability_store.go` 管安装和 Agent 绑定，`market_prompt_store.go` 管系统提示词模板、版本和变量，`market_scan.go` 管 scan helper，`market_util.go` 管可见性和 token 估算。

`internal/store` 的 Tool 部分按调用生命周期拆分：`tool_store.go` 只保留工具定义、调用日志、审批请求等类型和构造器，`tool_definition_store.go` 管内置工具和 MCP tool 同步，`tool_query_store.go` 管工具列表、Agent 绑定工具和单工具查找，`tool_log_store.go` 管 router log 与 call log，`tool_approval_store.go` 管审批请求创建和处理，`tool_scan.go` 管 scan helper，`tool_util.go` 管 owner 工具函数。

`internal/store` 的 Workspace 部分按生命周期拆分：`workspace_store.go` 只保留 workspace/file/event/command/quota 类型和构造器，`workspace_lifecycle_store.go` 管启用、策略更新、触达和销毁状态，`workspace_file_store.go` 管文件记录和事件，`workspace_command_store.go` 管命令运行记录，`workspace_quota_store.go` 管配额快照，`workspace_scan.go` 管 scan helper。

`internal/store` 的 Scheduled Job 部分按 job 与 run 拆分：`scheduled_job_store.go` 只保留心跳任务和运行记录类型，`scheduled_job_lifecycle_store.go` 管任务创建、更新、列表、到期扫描和状态更新，`scheduled_job_run_store.go` 管运行记录，`scheduled_job_scan.go` 管 scan helper。

`internal/store` 的 Agent Config 部分按配置本体和额外 LLM feature 拆分：`agent_config_store.go` 只保留 Agent 配置和 feature setting 类型，`agent_config_lifecycle_store.go` 管默认配置创建、读取和更新，`agent_config_feature_store.go` 管 feature setting 默认项和替换更新，`agent_config_scan.go` 管 scan helper，`agent_config_util.go` 管枚举归一化。

`internal/store` 的 Harness 部分按 Agent Turn 生命周期拆分：`harness_store.go` 只保留 turn/event/loop/validation/fallback 类型，`harness_turn_store.go` 管 turn 创建、启动、完成和读取，`harness_event_store.go` 管事件追加和列表，`harness_loop_store.go` 管 loop step，`harness_reliability_store.go` 管 validation/fallback 记录，`harness_scan.go` 管 scan helper。

后续运行命令建议：

```bash
go mod tidy
go run ./cmd/server
```

可选环境变量：

```bash
WORKSPACE_ROOT=./.workspaces
WORKSPACE_SANDBOX_IMAGE=freedinner-agent-sandbox:latest
WORKSPACE_DOCKER_BINARY=docker
WORKSPACE_PODMAN_BINARY=podman
WORKSPACE_NSJAIL_BINARY=nsjail
SCHEDULER_WORKER_ENABLED=true
SCHEDULER_POLL_INTERVAL=1m
CHANNEL_SENDER_ENABLED=true
CHANNEL_SENDER_INTERVAL=15s
CHANNEL_SENDER_BATCH_SIZE=20
```

本地开发默认使用 `./.workspaces` 和 `local_dir` sandbox。Linux 部署时可以把 `WORKSPACE_ROOT` 改成 `/var/lib/freedinner/workspaces`，再将用户 workspace 的 `sandbox_type` 配成 `docker`、`podman` 或 `nsjail`。这些 runtime 不会在本地开发时自动启动，只有对应用户启用对应 sandbox 类型时才会被调用。

## 当前已实现接口

健康检查：

```bash
curl http://localhost:8080/healthz
```

注册：

```bash
curl -sS -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"gexinyu","password":"password123","display_name":"Gexin"}'
```

登录：

```bash
curl -sS -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"gexinyu","password":"password123"}'
```

当前用户：

```bash
TOKEN="<登录返回的 access_token>"
curl -sS http://localhost:8080/api/v1/me \
  -H "Authorization: Bearer $TOKEN"
```

获取 Agent 配置：

```bash
curl -sS http://localhost:8080/api/v1/me/agent-config \
  -H "Authorization: Bearer $TOKEN"
```

更新 embedding 成本开关：

```bash
curl -sS -X PATCH http://localhost:8080/api/v1/me/agent-config \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"embedding_enabled":true,"embedding_cost_policy":{"mode":"manual","max_monthly_tokens":100000,"embed_public_knowledge":false}}'
```

开启某个额外 LLM 后台能力，例如自动压缩使用一个更便宜的 OpenAI-compatible provider 做摘要：

```bash
curl -sS -X PATCH http://localhost:8080/api/v1/me/agent-config \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"llm_feature_settings":[{"feature_key":"auto_compression_llm","enabled":true,"provider_id":"cheap-provider-id","model_override":"cheap-summary-model","temperature":0.2}]}'
```

默认情况下这些额外 LLM 能力全部关闭，自动压缩、Dreaming 和 Curator 会优先使用本地规则版实现，避免不知不觉消耗用户 token。额外 LLM 功能配置存储在 `agent_llm_feature_settings` 表中，每个 feature 可独立设置 `enabled`、`provider_id`、`model_override` 和 `temperature`。

新增模型供应商配置：

```bash
curl -sS -X POST http://localhost:8080/api/v1/me/model-providers \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"provider":"openai","display_name":"OpenAI 主账号","chat_base_url":"https://api.openai.com/v1","chat_api_key":"sk-...","default_chat_model":"gpt-4.1","embedding_base_url":"https://api.siliconflow.cn/v1","embedding_api_key":"sk-...","default_embedding_model":"Pro/BAAI/bge-m3","is_default":true}'
```

模型供应商配置是用户级别保存的。后端会加密保存 `chat_api_key` 和 `embedding_api_key`，接口不会返回明文；如果当前用户不开启 embedding，可以先不填 embedding 相关字段。

获取模型供应商配置：

```bash
curl -sS http://localhost:8080/api/v1/me/model-providers \
  -H "Authorization: Bearer $TOKEN"
```

导入知识库文档：

```bash
curl -sS -X POST http://localhost:8080/api/v1/knowledge-documents \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"项目背景","content":"FreeDinnerAgent 是一个具备长期记忆、工具调用和多入口接入能力的个人助理。","source_type":"manual","visibility":"private"}'
```

检索知识库：

```bash
curl -sS "http://localhost:8080/api/v1/knowledge-search?q=长期记忆&limit=5" \
  -H "Authorization: Bearer $TOKEN"
```

保存 Profile Memory：

```bash
curl -sS -X POST http://localhost:8080/api/v1/profile-memories \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"memory_type":"preference","title":"回答风格偏好","content":"用户喜欢回答先给结论，再给必要步骤。","evidence":"用户多次要求直接说重点。","importance":4,"confidence":0.9}'
```

检索 Profile Memory：

```bash
curl -sS "http://localhost:8080/api/v1/profile-memory-search?q=回答风格&limit=5" \
  -H "Authorization: Bearer $TOKEN"
```

预览本轮会加载的记忆上下文：

```bash
curl -sS "http://localhost:8080/api/v1/memory-context?conversation_id=<conversation_id>&q=根据知识库和我的偏好回答&max_memory_tokens=1200" \
  -H "Authorization: Bearer $TOKEN"
```

手动压缩当前对话：

```bash
curl -sS -X POST "http://localhost:8080/api/v1/conversations/<conversation_id>/compress" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"keep_recent_turns":8,"target_summary_type":"turn_window"}'
```

查看工具列表：

```bash
curl -sS http://localhost:8080/api/v1/tools \
  -H "Authorization: Bearer $TOKEN"
```

调用工具创建任务：

```bash
curl -sS -X POST http://localhost:8080/api/v1/tools/create_task/call \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"conversation_id":"<conversation_id>","arguments":{"title":"整理 Step 8 工具接口","priority":"high"}}'
```

创建心跳任务：

```bash
curl -sS -X POST http://localhost:8080/api/v1/scheduled-agent-jobs \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"每日简报","description":"工作日早上生成个人简报。","job_type":"daily_brief","schedule_kind":"weekly","timezone":"Asia/Shanghai","run_at_local_time":"08:00","weekdays":[1,2,3,4,5],"prompt_template":"请根据我的任务、记忆和最近对话生成今天的个人简报。","delivery_channel":"in_app"}'
```

查看心跳任务：

```bash
curl -sS "http://localhost:8080/api/v1/scheduled-agent-jobs?status=active" \
  -H "Authorization: Bearer $TOKEN"
```

手动触发心跳任务：

```bash
JOB_ID="<创建心跳任务返回的 id>"
curl -sS -X POST "http://localhost:8080/api/v1/scheduled-agent-jobs/$JOB_ID/run-now" \
  -H "Authorization: Bearer $TOKEN"
```

查看心跳任务运行记录：

```bash
curl -sS "http://localhost:8080/api/v1/scheduled-agent-jobs/$JOB_ID/runs" \
  -H "Authorization: Bearer $TOKEN"
```

查看推荐心跳任务模板：

```bash
curl -sS http://localhost:8080/api/v1/scheduled-agent-job-templates \
  -H "Authorization: Bearer $TOKEN"
```

更新/暂停/恢复/删除心跳任务：

```bash
curl -sS -X PATCH "http://localhost:8080/api/v1/scheduled-agent-jobs/$JOB_ID" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"run_at_local_time":"09:00","weekdays":[1,2,3,4,5]}'

curl -sS -X POST "http://localhost:8080/api/v1/scheduled-agent-jobs/$JOB_ID/pause" \
  -H "Authorization: Bearer $TOKEN"

curl -sS -X POST "http://localhost:8080/api/v1/scheduled-agent-jobs/$JOB_ID/resume" \
  -H "Authorization: Bearer $TOKEN"

curl -sS -X DELETE "http://localhost:8080/api/v1/scheduled-agent-jobs/$JOB_ID" \
  -H "Authorization: Bearer $TOKEN"
```

查看可用渠道 Provider：

```bash
curl -sS http://localhost:8080/api/v1/channel-providers \
  -H "Authorization: Bearer $TOKEN"
```

Channel Adapter 和普通 Web Chat 的触发方式不同：

- Web Chat 是用户主动对话，用户在某个 conversation 里发送 query 才触发 Agent Loop。
- Channel Adapter 是监听入口，外部 QQ 私聊、群聊 @ 或关键字命中后，由 webhook 事件触发 Agent Loop。
- 前端建议单独做 Channels 页面管理连接、策略、inbox、outbox 和审批；不要把 Channel connection 当成普通“新建对话”入口。
- 每个 `channel_connection` 默认对应一个专用监听/主控会话；外部私聊、群聊等 scope 通过 `external_conversations` 映射到本地 conversation，并在 UI 上归属该 Channel connection。
- 微信、Telegram、Discord、飞书等具体 Adapter 归入高级项，只保留抽象；当前可运行验证入口是 NapCatQQ / OneBot。
- approved outbox 可以通过显式接口发送，也可以由后台 sender worker 自动发送。

创建 NapCatQQ 渠道连接：

```bash
NAPCAT_PROVIDER_ID="<channel-providers 里 name=napcatqq 的 id>"
curl -sS -X POST http://localhost:8080/api/v1/me/channel-connections \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"provider_id":"'"$NAPCAT_PROVIDER_ID"'","display_name":"本地 NapCatQQ","external_account_id":"你的机器人 QQ 号","external_account_name":"FreeDinnerBot","config":{"access_token":"napcat-token","webhook_secret":"hook-secret","bot_qq":"你的机器人 QQ 号"},"endpoints":[{"endpoint_type":"message_api","display_name":"NapCat HTTP API","direction":"outbound","transport":"http","url":"http://127.0.0.1:3000","config":{"access_token":"napcat-token"}},{"endpoint_type":"event_stream","display_name":"NapCat HTTP SSE","direction":"inbound","transport":"http_sse","url":"http://127.0.0.1:3000/sse","config":{"access_token":"napcat-token"}},{"endpoint_type":"webhook_callback","display_name":"FreeDinnerAgent Webhook","direction":"inbound","transport":"http","url":"http://127.0.0.1:8080/api/v1/channels/<connection_id>/webhook","config":{"webhook_secret":"hook-secret"}}]}'
```

NapCatQQ / OneBot Webhook 地址：

```text
POST http://localhost:8080/api/v1/channels/<connection_id>/webhook
Header: X-FreeDinner-Webhook-Secret: hook-secret
```

NapCat 部署与 HTTP SSE 服务器、HTTP 客户端配置说明见 [NAPCATQQ.md](NAPCATQQ.md)。`message_api` / `event_stream` / `webhook_callback` 的 URL 只要求被对应服务进程访问到，具体使用公网、内网、localhost 或隧道由部署方式决定。

本地模拟一条 QQ 私聊消息：

```bash
CONNECTION_ID="<创建渠道连接返回的 id>"
curl -sS -X POST "http://localhost:8080/api/v1/channels/$CONNECTION_ID/webhook" \
  -H "Content-Type: application/json" \
  -H "X-FreeDinner-Webhook-Secret: hook-secret" \
  -d '{"post_type":"message","message_type":"private","message_id":123456,"user_id":10001,"raw_message":"你好，小饭","sender":{"nickname":"测试好友"}}'
```

群聊默认只有 @ 机器人时触发，且外发消息默认进入 `pending`：

```bash
curl -sS -X POST "http://localhost:8080/api/v1/channels/$CONNECTION_ID/webhook" \
  -H "Content-Type: application/json" \
  -H "X-FreeDinner-Webhook-Secret: hook-secret" \
  -d '{"post_type":"message","message_type":"group","message_id":223344,"group_id":88888,"user_id":10002,"raw_message":"[CQ:at,qq=你的机器人 QQ 号] 帮我总结一下","sender":{"card":"群友A"}}'
```

查看渠道 inbox / outbox：

```bash
curl -sS "http://localhost:8080/api/v1/me/channel-connections/$CONNECTION_ID/inbox-events" \
  -H "Authorization: Bearer $TOKEN"

curl -sS "http://localhost:8080/api/v1/me/channel-connections/$CONNECTION_ID/outbox-messages" \
  -H "Authorization: Bearer $TOKEN"
```

审批或取消群聊 outbox 草稿：

```bash
OUTBOX_ID="<outbox message id>"
curl -sS -X POST "http://localhost:8080/api/v1/channel-outbox-messages/$OUTBOX_ID/approve" \
  -H "Authorization: Bearer $TOKEN"

curl -sS -X POST "http://localhost:8080/api/v1/channel-outbox-messages/$OUTBOX_ID/cancel" \
  -H "Authorization: Bearer $TOKEN"
```

查看能力市场并创建系统提示词模板：

```bash
curl -sS "http://localhost:8080/api/v1/marketplace-items?item_type=system_prompt_template" \
  -H "Authorization: Bearer $TOKEN"

curl -sS -X POST http://localhost:8080/api/v1/system-prompt-templates \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"research_assistant","display_name":"研究助理","description":"适合论文阅读和课题整理。","category":"research","tags":["research"],"visibility":"private","content":"你是 {agent_name}，请用 {language} 回答，并优先保持严谨。","variables":[{"name":"agent_name","display_name":"Agent 名称","value_type":"string","required":true,"default_value":"小饭"},{"name":"language","display_name":"回答语言","value_type":"enum","required":true,"default_value":"中文","allowed_values":["中文","English"]}]}'
```

预览并绑定系统提示词模板版本：

```bash
TEMPLATE_VERSION_ID="<创建模板返回的 version.id>"
curl -sS -X POST http://localhost:8080/api/v1/system-prompt-templates/preview \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"version_id":"'"$TEMPLATE_VERSION_ID"'","variables":{"agent_name":"小饭","language":"中文"},"override":"回答时先给结论。"}'

curl -sS -X POST http://localhost:8080/api/v1/agent-capability-bindings \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"capability_type":"system_prompt_template","capability_ref_id":"'"$TEMPLATE_VERSION_ID"'","load_mode":"full","priority":100}'
```

启用 Workspace：

```bash
curl -sS -X POST http://localhost:8080/api/v1/me/workspace \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"sandbox_type":"local_dir","network_policy":"disabled","max_disk_bytes":1073741824,"max_single_file_bytes":52428800,"cpu_limit":"1.0","memory_limit_bytes":536870912}'
```

查看 Workspace 状态：

```bash
curl -sS http://localhost:8080/api/v1/me/workspace \
  -H "Authorization: Bearer $TOKEN"
```

写入 Workspace 文件：

```bash
curl -sS -X PUT http://localhost:8080/api/v1/me/workspace/files/content \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"path":"/notes/todo.md","content":"今天继续完成 FreeDinnerAgent。"}'
```

列出 Workspace 文件：

```bash
curl -sS "http://localhost:8080/api/v1/me/workspace/files?path=/notes" \
  -H "Authorization: Bearer $TOKEN"
```

读取 Workspace 文件：

```bash
curl -sS "http://localhost:8080/api/v1/me/workspace/files/content?path=/notes/todo.md" \
  -H "Authorization: Bearer $TOKEN"
```

执行 Workspace 白名单命令：

```bash
curl -sS -X POST http://localhost:8080/api/v1/me/workspace/commands \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"command":"cat","args":["notes/todo.md"],"working_dir":"/","timeout_seconds":5}'
```

查看命令历史：

```bash
curl -sS "http://localhost:8080/api/v1/me/workspace/commands?limit=20" \
  -H "Authorization: Bearer $TOKEN"
```

更新 Workspace 策略：

```bash
curl -sS -X PATCH http://localhost:8080/api/v1/me/workspace \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"sandbox_type":"local_dir","network_policy":"disabled","max_command_seconds":10}'
```

销毁 Workspace 记录：

```bash
curl -sS -X DELETE "http://localhost:8080/api/v1/me/workspace?remove_files=false" \
  -H "Authorization: Bearer $TOKEN"
```

创建会话：

```bash
curl -sS -X POST http://localhost:8080/api/v1/conversations \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"今天的计划"}'
```

发送消息：

```bash
CONVERSATION_ID="<创建会话返回的 id>"
curl -sS -X POST "http://localhost:8080/api/v1/conversations/$CONVERSATION_ID/messages" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"content":"我明天下午三点要交报告，帮我记一下。"}'
```

发送消息会读取当前用户默认模型供应商配置。当前阶段已接入 OpenAI Responses API，并兼容常见 OpenAI Chat Completions 格式的第三方网关。配置第三方网关时，`chat_base_url` 可以填写类似 `https://example.com/openai/v1` 或完整的 `https://example.com/openai/v1/chat/completions`。如果还没有配置 provider，会返回 `MODEL_PROVIDER_REQUIRED`。默认 provider 重试后仍失败时，后端会尝试切换到同用户其它 active/openai-compatible provider，并写入 `provider_fallback` 事件。一次发送会同步创建 `agent_turns`，成功响应里的 `turn_id` 可用于查看本轮 Agent 执行轨迹。

查看本轮 Agent 执行事件：

```bash
TURN_ID="<发送消息返回的 turn_id>"
curl -sS "http://localhost:8080/api/v1/conversations/$CONVERSATION_ID/agent-events?turn_id=$TURN_ID" \
  -H "Authorization: Bearer $TOKEN"
```

查看本轮 Agent Loop 步骤：

```bash
curl -sS "http://localhost:8080/api/v1/conversations/$CONVERSATION_ID/agent-turns/$TURN_ID/loop-steps" \
  -H "Authorization: Bearer $TOKEN"
```

获取会话消息：

```bash
curl -sS "http://localhost:8080/api/v1/conversations/$CONVERSATION_ID/messages" \
  -H "Authorization: Bearer $TOKEN"
```
