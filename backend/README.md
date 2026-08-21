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
