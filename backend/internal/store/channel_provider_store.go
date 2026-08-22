package store

import (
	"context"

	"github.com/google/uuid"
)

func (s *ChannelStore) EnsureBuiltinProviders(ctx context.Context) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO channel_provider_definitions (
			id, name, display_name, description, provider_type, adapter_type,
			inbound_modes, outbound_modes, config_schema, default_policy, visibility, status, metadata
		)
		VALUES (
			$1, 'napcatqq', 'NapCatQQ', '基于 OneBot/NapCatQQ 的 QQ 私聊与群聊入口。', 'qq', 'http_webhook',
			ARRAY['http_webhook', 'http_sse', 'websocket'], ARRAY['send_message'],
			'{"type":"object","properties":{"config":{"type":"object","description":"Connection-level secrets and identity. Endpoint URLs are stored in channel_connection_endpoints.","properties":{"access_token":{"type":"string","description":"Permanent NapCat access token shared by endpoint configs when needed"},"webhook_secret":{"type":"string","description":"Permanent secret/token sent by NapCat HTTP client"},"bot_qq":{"type":"string","description":"QQ account logged into NapCat"}}},"endpoints":{"type":"array","description":"Provider endpoint templates. Users can configure concrete URLs per connection.","items":{"type":"object","properties":{"endpoint_type":{"type":"string","enum":["message_api","event_stream","webhook_callback"]},"direction":{"type":"string","enum":["inbound","outbound","bidirectional"]},"transport":{"type":"string","enum":["http","http_sse","websocket","grpc","custom"]},"url":{"type":"string"},"config":{"type":"object","description":"Endpoint-level secret config, such as access_token or webhook_secret."}}}}}}'::jsonb,
			'{"private_chat":{"mode":"auto_reply"},"group_chat":{"mode":"mention_only","require_approval_for_outbound":true}}'::jsonb,
			'public', 'active', '{}'::jsonb
		)
		ON CONFLICT (name) DO UPDATE SET
			display_name = EXCLUDED.display_name,
			description = EXCLUDED.description,
			inbound_modes = EXCLUDED.inbound_modes,
			outbound_modes = EXCLUDED.outbound_modes,
			config_schema = EXCLUDED.config_schema,
			default_policy = EXCLUDED.default_policy,
			status = 'active',
			updated_at = NOW()
	`, uuid.NewString())
	return err
}

func (s *ChannelStore) ListProviders(ctx context.Context) ([]ChannelProviderDefinition, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, name, display_name, description, provider_type, adapter_type, inbound_modes,
			outbound_modes, config_schema, default_policy, visibility, status, metadata, created_at, updated_at
		FROM channel_provider_definitions
		WHERE status = 'active' AND visibility = 'public'
		ORDER BY display_name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	providers := make([]ChannelProviderDefinition, 0)
	for rows.Next() {
		provider, err := scanChannelProvider(rows)
		if err != nil {
			return nil, err
		}
		providers = append(providers, provider)
	}
	return providers, rows.Err()
}
