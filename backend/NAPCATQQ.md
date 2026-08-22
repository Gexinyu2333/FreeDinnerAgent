# NapCatQQ / OneBot deployment notes

NapCatQQ is the first concrete Channel Adapter for FreeDinnerAgent. It should run as an external service. Do not vendor the NapCat source code into this repository.

## Required NapCat setup

On the machine that runs NapCat:

1. Install and start NapCatQQ.
2. Log in with the QQ account that will act as the Agent entrance.
3. Enable the NapCat HTTP / SSE server.
   - FreeDinnerAgent uses this endpoint to call OneBot APIs such as `/send_msg`.
   - The same server can later be used by a worker to listen to SSE events.
4. Enable the NapCat HTTP client.
   - Configure it to POST events to FreeDinnerAgent:
     `http://<freedinner-host>:8080/api/v1/channels/<connection_id>/webhook`
   - Add the same permanent token/secret that is stored in the Channel connection.

## FreeDinnerAgent Channel endpoints

The Channels page stores URL-like settings as endpoint rows in `channel_connection_endpoints`:

- `message_api`: NapCat HTTP API base URL used by FreeDinnerAgent to call `/send_msg`.
- `event_stream`: NapCat HTTP SSE server URL reserved for event streaming workers.
- `webhook_callback`: FreeDinnerAgent webhook URL configured in the NapCat HTTP client.

Sensitive values stay encrypted in `channel_connections.encrypted_config`:

- `access_token`: permanent NapCat access token.
- `webhook_secret`: permanent token sent by NapCat HTTP client in `X-FreeDinner-Webhook-Secret`.
- `bot_qq`: QQ account logged into NapCat. This is also stored as `external_account_id` when provided.

Endpoint-level tokens can also be stored in each endpoint's encrypted config. For NapCat, the frontend writes the access token to `message_api` and `event_stream`, and the webhook secret to `webhook_callback`.

## Deployment topology

Use URLs that are reachable from the process that needs them:

- `message_api` and `event_stream` must be reachable from the FreeDinnerAgent backend.
- `webhook_callback` must be reachable from the NapCat HTTP client.

They can be localhost URLs, private network URLs, public IPs, domains, or tunnel URLs depending on how the two services are deployed.

Tokens are treated as long-lived for the current MVP so they match NapCat's permanent HTTP server/client token setup.

## Existing database migration

If your local database was initialized before these NapCat fields existed, run:

```bash
/opt/homebrew/opt/postgresql@17/bin/psql -U freedinner -d freedinner_agent -v ON_ERROR_STOP=1 -c "CREATE TABLE IF NOT EXISTS channel_connection_endpoints (id UUID PRIMARY KEY, user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE, channel_connection_id UUID NOT NULL REFERENCES channel_connections(id) ON DELETE CASCADE, endpoint_type VARCHAR(80) NOT NULL, display_name VARCHAR(160) NOT NULL, direction VARCHAR(32) NOT NULL CHECK (direction IN ('inbound', 'outbound', 'bidirectional')), transport VARCHAR(40) NOT NULL CHECK (transport IN ('http', 'http_sse', 'websocket', 'grpc', 'custom')), url TEXT NOT NULL, encrypted_config JSONB NOT NULL DEFAULT '{}'::jsonb, status VARCHAR(32) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused', 'revoked', 'deleted')), metadata JSONB NOT NULL DEFAULT '{}'::jsonb, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), UNIQUE (channel_connection_id, endpoint_type)); CREATE INDEX IF NOT EXISTS idx_channel_connection_endpoints_connection_status ON channel_connection_endpoints(channel_connection_id, status); CREATE INDEX IF NOT EXISTS idx_channel_connection_endpoints_type_status ON channel_connection_endpoints(endpoint_type, status); ALTER TABLE channel_connections DROP COLUMN IF EXISTS adapter_endpoint_url, DROP COLUMN IF EXISTS adapter_sse_url, DROP COLUMN IF EXISTS webhook_callback_url;"
```

Fresh databases created from `database/init.sql` already contain these columns.
