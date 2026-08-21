# Backend

本目录用于存放 FreeDinnerAgent 的 Go 后端服务。

计划技术栈：

- Go
- Gin
- PostgreSQL Driver
- LLM API Client

核心模块规划：

- `cmd/server`：服务启动入口
- `internal/api`：HTTP 路由和请求处理
- `internal/agent`：Agent 编排、上下文管理、工具调用决策
- `internal/memory`：记忆写入、检索和管理
- `internal/tool`：工具注册、参数校验、执行和降级处理
- `internal/store`：数据库访问层
- `internal/config`：环境变量和配置加载

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

创建 NapCatQQ 渠道连接：

```bash
NAPCAT_PROVIDER_ID="<channel-providers 里 name=napcatqq 的 id>"
curl -sS -X POST http://localhost:8080/api/v1/me/channel-connections \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"provider_id":"'"$NAPCAT_PROVIDER_ID"'","display_name":"本地 NapCatQQ","external_account_id":"你的机器人 QQ 号","external_account_name":"FreeDinnerBot","config":{"endpoint":"http://127.0.0.1:3000","access_token":"napcat-token","webhook_secret":"hook-secret"}}'
```

NapCatQQ / OneBot Webhook 地址：

```text
POST http://localhost:8080/api/v1/channels/<connection_id>/webhook
Header: X-FreeDinner-Webhook-Secret: hook-secret
```

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

发送消息会读取当前用户默认模型供应商配置。当前阶段已接入 OpenAI Responses API，并兼容常见 OpenAI Chat Completions 格式的第三方网关。配置第三方网关时，`chat_base_url` 可以填写类似 `https://example.com/openai/v1` 或完整的 `https://example.com/openai/v1/chat/completions`。如果还没有配置 provider，会返回 `MODEL_PROVIDER_REQUIRED`。一次发送会同步创建 `agent_turns`，成功响应里的 `turn_id` 可用于查看本轮 Agent 执行轨迹。

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
