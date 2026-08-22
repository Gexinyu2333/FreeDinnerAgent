# NapCat / OneBot 接入说明

NapCat 是 FreeDinnerAgent 当前第一条已经跑通的外部 Channel 入口。NapCat 本身作为独立服务部署，不需要把 NapCat 源码拉进本项目。

当前已验证的链路：

```text
QQ群消息
-> NapCat HTTP Client
-> FreeDinnerAgent webhook
-> Agent Loop
-> channel_outbox_messages
-> sender worker
-> NapCat /send_msg
-> QQ 群回复
```

## NapCat 侧配置

在运行 NapCat 的机器上完成以下配置：

1. 安装并启动 NapCat。
2. 登录作为 Agent 入口的 QQ 账号。
3. 开启 HTTP SSE Server。
4. 开启 HTTP Client，把事件上报到 FreeDinnerAgent 的 webhook。

HTTP SSE Server 推荐配置：

```text
Host: 0.0.0.0
Port: 3000
Token: 自己设置的长期 token
消息格式: Array
```

如果 FreeDinnerAgent 后端需要从其它机器访问 NapCat，Host 不要只填 `127.0.0.1`，否则只能 NapCat 所在机器本机访问。

HTTP Client 的 URL 需要填完整 webhook 地址：

```text
http://<freedinner-host>:8080/api/v1/channels/<connection_id>/webhook?token=<webhook_secret>
```

当前后端 webhook 支持从这些位置读取 secret：

- `X-FreeDinner-Webhook-Secret`
- `X-Access-Token`
- `X-Token`
- `Authorization: Bearer <token>`
- `Authorization: Token <token>`
- `Authorization: <token>`
- URL query：`?token=<token>`、`?access_token=<token>`、`?secret=<token>`

实际调试时，NapCat HTTP Client 的 Token 字段可能不会按预期 header 传递。最稳的方式是在 URL 里加 `?token=...`。

## FreeDinnerAgent Endpoint 模型

Channel 的 URL 类配置统一写入 `channel_connection_endpoints`，不要给 `channel_connections` 主表增加平台专属 URL 字段。这样以后接 WeChat、Telegram、Discord、飞书等入口时，仍然复用同一套结构。

NapCat 连接通常包含三个 endpoint：

- `message_api`：NapCat HTTP API base URL，FreeDinnerAgent 用它调用 `/send_msg`。
- `event_stream`：NapCat HTTP SSE 地址，预留给后续 SSE worker 监听。
- `webhook_callback`：NapCat HTTP Client 上报事件时调用的 FreeDinnerAgent webhook URL。

敏感值会加密存储：

- `channel_connections.encrypted_config`：连接级 token、webhook secret、bot QQ。
- `channel_connection_endpoints.encrypted_config`：endpoint 级 token 或 secret。

当前 MVP 把这些 token 视为长期有效，和 NapCat 的 HTTP Server / HTTP Client token 设置保持一致。

## 本机开发调试

如果 NapCat 在云服务器上，FreeDinnerAgent 后端在你的 Mac 本机上，家用 Wi-Fi 通常不能直接让云服务器访问 Mac 的 `localhost:8080`。可以用 SSH 反向隧道，不需要额外安装内网穿透工具。

在 Mac 上先启动后端：

```bash
cd /Users/gexinyu/GolandProjects/FreeDinnerAgent/backend
go run ./cmd/server
```

再开另一个终端建立 SSH 反向隧道：

```bash
ssh -v -N -o ExitOnForwardFailure=yes -R 18080:127.0.0.1:8080 root@<napcat-server-ip>
```

成功后，NapCat 所在机器访问：

```bash
curl -i --max-time 5 http://127.0.0.1:18080/healthz
```

应该返回：

```json
{"data":{"env":"development","status":"ok"},"error":null}
```

此时 NapCat HTTP Client URL 填：

```text
http://127.0.0.1:18080/api/v1/channels/<connection_id>/webhook?token=<webhook_secret>
```

## 出站发送验证

先确认 NapCat HTTP API 可访问：

```bash
curl -i http://<napcat-host>:3000/
```

正常会返回类似：

```json
{"status":"ok","retcode":0,"data":{},"message":"NapCat4 Is Running"}
```

直连发送群消息：

```bash
curl -i -H "Authorization: Bearer <napcat-token>" \
  -H "Content-Type: application/json" \
  -d '{"message_type":"group","group_id":"<group_id>","message":"FreeDinnerAgent NapCat 直连发送测试"}' \
  http://<napcat-host>:3000/send_msg
```

成功时会返回 `retcode = 0` 和 `message_id`。

## 入站监听验证

在 QQ 群里发送一条测试消息后，查看后端日志应出现：

```text
POST /api/v1/channels/<connection_id>/webhook 200
```

查看数据库：

```bash
psql -U freedinner -d freedinner_agent -c "
SELECT normalized_text, external_sender_id, should_trigger_agent, status, received_at
FROM channel_inbox_events
WHERE channel_connection_id = '<connection_id>'
ORDER BY received_at DESC
LIMIT 5;
"
```

如果自动触发 Agent，并且 sender worker 正常发送，应能在 outbox 表看到 `sent`：

```bash
psql -U freedinner -d freedinner_agent -c "
SELECT content, status, external_message_id, error_message, sent_at
FROM channel_outbox_messages
WHERE channel_connection_id = '<connection_id>'
ORDER BY created_at DESC
LIMIT 5;
"
```

## 已有数据库迁移

新数据库直接执行 `database/init.sql` 即可，已经包含 `channel_connection_endpoints`。

如果本地数据库是在 endpoint 表加入之前初始化的，可以执行：

```bash
/opt/homebrew/opt/postgresql@17/bin/psql -U freedinner -d freedinner_agent -v ON_ERROR_STOP=1 -c "CREATE TABLE IF NOT EXISTS channel_connection_endpoints (id UUID PRIMARY KEY, user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE, channel_connection_id UUID NOT NULL REFERENCES channel_connections(id) ON DELETE CASCADE, endpoint_type VARCHAR(80) NOT NULL, display_name VARCHAR(160) NOT NULL, direction VARCHAR(32) NOT NULL CHECK (direction IN ('inbound', 'outbound', 'bidirectional')), transport VARCHAR(40) NOT NULL CHECK (transport IN ('http', 'http_sse', 'websocket', 'grpc', 'custom')), url TEXT NOT NULL, encrypted_config JSONB NOT NULL DEFAULT '{}'::jsonb, status VARCHAR(32) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused', 'revoked', 'deleted')), metadata JSONB NOT NULL DEFAULT '{}'::jsonb, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), UNIQUE (channel_connection_id, endpoint_type)); CREATE INDEX IF NOT EXISTS idx_channel_connection_endpoints_connection_status ON channel_connection_endpoints(channel_connection_id, status); CREATE INDEX IF NOT EXISTS idx_channel_connection_endpoints_type_status ON channel_connection_endpoints(endpoint_type, status); ALTER TABLE channel_connections DROP COLUMN IF EXISTS adapter_endpoint_url, DROP COLUMN IF EXISTS adapter_sse_url, DROP COLUMN IF EXISTS webhook_callback_url;"
```

