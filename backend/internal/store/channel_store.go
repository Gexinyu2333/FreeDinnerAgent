package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ChannelProviderDefinition struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	DisplayName   string          `json:"display_name"`
	Description   string          `json:"description"`
	ProviderType  string          `json:"provider_type"`
	AdapterType   string          `json:"adapter_type"`
	InboundModes  []string        `json:"inbound_modes"`
	OutboundModes []string        `json:"outbound_modes"`
	ConfigSchema  json.RawMessage `json:"config_schema"`
	DefaultPolicy json.RawMessage `json:"default_policy"`
	Visibility    string          `json:"visibility"`
	Status        string          `json:"status"`
	Metadata      json.RawMessage `json:"metadata"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type ChannelConnection struct {
	ID                  string          `json:"id"`
	UserID              string          `json:"user_id"`
	ProviderID          string          `json:"provider_id"`
	DisplayName         string          `json:"display_name"`
	ExternalAccountID   *string         `json:"external_account_id"`
	ExternalAccountName *string         `json:"external_account_name"`
	EncryptedConfig     json.RawMessage `json:"-"`
	Status              string          `json:"status"`
	LastHealthStatus    *string         `json:"last_health_status"`
	LastEventAt         *time.Time      `json:"last_event_at"`
	LastCheckedAt       *time.Time      `json:"last_checked_at"`
	Metadata            json.RawMessage `json:"metadata"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

type PublicChannelConnection struct {
	ID                  string     `json:"id"`
	UserID              string     `json:"user_id"`
	ProviderID          string     `json:"provider_id"`
	DisplayName         string     `json:"display_name"`
	ExternalAccountID   *string    `json:"external_account_id"`
	ExternalAccountName *string    `json:"external_account_name"`
	HasConfig           bool       `json:"has_config"`
	Status              string     `json:"status"`
	LastHealthStatus    *string    `json:"last_health_status"`
	LastEventAt         *time.Time `json:"last_event_at"`
	LastCheckedAt       *time.Time `json:"last_checked_at"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type ChannelPolicy struct {
	ID                         string          `json:"id"`
	UserID                     string          `json:"user_id"`
	ChannelConnectionID        string          `json:"channel_connection_id"`
	ScopeType                  string          `json:"scope_type"`
	ExternalScopeID            *string         `json:"external_scope_id"`
	Mode                       string          `json:"mode"`
	TriggerKeywords            []string        `json:"trigger_keywords"`
	AllowMemoryWrite           bool            `json:"allow_memory_write"`
	AllowToolUse               bool            `json:"allow_tool_use"`
	RequireApprovalForOutbound bool            `json:"require_approval_for_outbound"`
	RateLimitPerMinute         int             `json:"rate_limit_per_minute"`
	QuietHours                 json.RawMessage `json:"quiet_hours"`
	Status                     string          `json:"status"`
	Metadata                   json.RawMessage `json:"metadata"`
	CreatedAt                  time.Time       `json:"created_at"`
	UpdatedAt                  time.Time       `json:"updated_at"`
}

type ExternalConversation struct {
	ID                       string          `json:"id"`
	UserID                   string          `json:"user_id"`
	ChannelConnectionID      string          `json:"channel_connection_id"`
	ConversationID           string          `json:"conversation_id"`
	ExternalConversationID   string          `json:"external_conversation_id"`
	ExternalConversationType string          `json:"external_conversation_type"`
	ExternalTitle            *string         `json:"external_title"`
	LastMessageAt            *time.Time      `json:"last_message_at"`
	Status                   string          `json:"status"`
	Metadata                 json.RawMessage `json:"metadata"`
	CreatedAt                time.Time       `json:"created_at"`
	UpdatedAt                time.Time       `json:"updated_at"`
}

type ChannelInboxEvent struct {
	ID                     string          `json:"id"`
	UserID                 string          `json:"user_id"`
	ChannelConnectionID    string          `json:"channel_connection_id"`
	ExternalConversationID *string         `json:"external_conversation_id"`
	ConversationID         *string         `json:"conversation_id"`
	MessageID              *string         `json:"message_id"`
	EventType              string          `json:"event_type"`
	ExternalEventID        *string         `json:"external_event_id"`
	ExternalSenderID       *string         `json:"external_sender_id"`
	ExternalSenderName     *string         `json:"external_sender_name"`
	RawPayload             json.RawMessage `json:"raw_payload"`
	NormalizedText         *string         `json:"normalized_text"`
	ShouldTriggerAgent     bool            `json:"should_trigger_agent"`
	TriggerReason          *string         `json:"trigger_reason"`
	Status                 string          `json:"status"`
	ReceivedAt             time.Time       `json:"received_at"`
	ProcessedAt            *time.Time      `json:"processed_at"`
}

type ChannelOutboxMessage struct {
	ID                     string          `json:"id"`
	UserID                 string          `json:"user_id"`
	ChannelConnectionID    string          `json:"channel_connection_id"`
	ExternalConversationID *string         `json:"external_conversation_id"`
	ConversationID         *string         `json:"conversation_id"`
	AgentTurnID            *string         `json:"agent_turn_id"`
	ReplyToInboxEventID    *string         `json:"reply_to_inbox_event_id"`
	MessageType            string          `json:"message_type"`
	Content                string          `json:"content"`
	Payload                json.RawMessage `json:"payload"`
	RequiresApproval       bool            `json:"requires_approval"`
	Status                 string          `json:"status"`
	ExternalMessageID      *string         `json:"external_message_id"`
	ErrorMessage           *string         `json:"error_message"`
	CreatedAt              time.Time       `json:"created_at"`
	ApprovedAt             *time.Time      `json:"approved_at"`
	SentAt                 *time.Time      `json:"sent_at"`
}

type ChannelConnectionCreate struct {
	UserID              string
	ProviderID          string
	DisplayName         string
	ExternalAccountID   *string
	ExternalAccountName *string
	EncryptedConfig     json.RawMessage
	Metadata            json.RawMessage
}

type ChannelPolicyUpsert struct {
	UserID                     string
	ChannelConnectionID        string
	ScopeType                  string
	ExternalScopeID            *string
	Mode                       string
	TriggerKeywords            []string
	AllowMemoryWrite           bool
	AllowToolUse               bool
	RequireApprovalForOutbound bool
	RateLimitPerMinute         int
	QuietHours                 json.RawMessage
	Metadata                   json.RawMessage
}

type ChannelStore struct {
	db *pgxpool.Pool
}

func NewChannelStore(db *pgxpool.Pool) *ChannelStore {
	return &ChannelStore{db: db}
}

func (s *ChannelStore) EnsureBuiltinProviders(ctx context.Context) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO channel_provider_definitions (
			id, name, display_name, description, provider_type, adapter_type,
			inbound_modes, outbound_modes, config_schema, default_policy, visibility, status, metadata
		)
		VALUES (
			$1, 'napcatqq', 'NapCatQQ', '基于 OneBot/NapCatQQ 的 QQ 私聊与群聊入口。', 'qq', 'http_webhook',
			ARRAY['http_webhook', 'websocket'], ARRAY['send_message'],
			'{"type":"object","properties":{"endpoint":{"type":"string"},"access_token":{"type":"string"},"webhook_secret":{"type":"string"}}}'::jsonb,
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
	var providers []ChannelProviderDefinition
	for rows.Next() {
		provider, err := scanChannelProvider(rows)
		if err != nil {
			return nil, err
		}
		providers = append(providers, provider)
	}
	return providers, rows.Err()
}

func (s *ChannelStore) CreateConnection(ctx context.Context, input ChannelConnectionCreate) (ChannelConnection, error) {
	if len(input.EncryptedConfig) == 0 {
		input.EncryptedConfig = json.RawMessage(`{}`)
	}
	if len(input.Metadata) == 0 {
		input.Metadata = json.RawMessage(`{}`)
	}
	query := `
		INSERT INTO channel_connections (
			id, user_id, provider_id, display_name, external_account_id, external_account_name,
			encrypted_config, metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, user_id, provider_id, display_name, external_account_id, external_account_name,
			encrypted_config, status, last_health_status, last_event_at, last_checked_at, metadata,
			created_at, updated_at
	`
	return scanChannelConnection(s.db.QueryRow(ctx, query, uuid.NewString(), input.UserID, input.ProviderID,
		input.DisplayName, input.ExternalAccountID, input.ExternalAccountName, input.EncryptedConfig, input.Metadata))
}

func (s *ChannelStore) ListConnections(ctx context.Context, userID string) ([]ChannelConnection, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, provider_id, display_name, external_account_id, external_account_name,
			encrypted_config, status, last_health_status, last_event_at, last_checked_at, metadata,
			created_at, updated_at
		FROM channel_connections
		WHERE user_id = $1 AND status <> 'deleted'
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var connections []ChannelConnection
	for rows.Next() {
		connection, err := scanChannelConnection(rows)
		if err != nil {
			return nil, err
		}
		connections = append(connections, connection)
	}
	return connections, rows.Err()
}

func (s *ChannelStore) FindConnectionByID(ctx context.Context, connectionID string) (ChannelConnection, error) {
	return scanChannelConnection(s.db.QueryRow(ctx, `
		SELECT id, user_id, provider_id, display_name, external_account_id, external_account_name,
			encrypted_config, status, last_health_status, last_event_at, last_checked_at, metadata,
			created_at, updated_at
		FROM channel_connections
		WHERE id = $1 AND status <> 'deleted'
	`, connectionID))
}

func (s *ChannelStore) FindUserConnectionByID(ctx context.Context, userID, connectionID string) (ChannelConnection, error) {
	return scanChannelConnection(s.db.QueryRow(ctx, `
		SELECT id, user_id, provider_id, display_name, external_account_id, external_account_name,
			encrypted_config, status, last_health_status, last_event_at, last_checked_at, metadata,
			created_at, updated_at
		FROM channel_connections
		WHERE id = $1 AND user_id = $2 AND status <> 'deleted'
	`, connectionID, userID))
}

func (s *ChannelStore) UpsertPolicy(ctx context.Context, input ChannelPolicyUpsert) (ChannelPolicy, error) {
	if len(input.QuietHours) == 0 {
		input.QuietHours = json.RawMessage(`{}`)
	}
	if len(input.Metadata) == 0 {
		input.Metadata = json.RawMessage(`{}`)
	}
	return scanChannelPolicy(s.db.QueryRow(ctx, `
		INSERT INTO channel_policies (
			id, user_id, channel_connection_id, scope_type, external_scope_id, mode, trigger_keywords,
			allow_memory_write, allow_tool_use, require_approval_for_outbound, rate_limit_per_minute,
			quiet_hours, metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (channel_connection_id, scope_type, external_scope_id) DO UPDATE SET
			mode = EXCLUDED.mode,
			trigger_keywords = EXCLUDED.trigger_keywords,
			allow_memory_write = EXCLUDED.allow_memory_write,
			allow_tool_use = EXCLUDED.allow_tool_use,
			require_approval_for_outbound = EXCLUDED.require_approval_for_outbound,
			rate_limit_per_minute = EXCLUDED.rate_limit_per_minute,
			quiet_hours = EXCLUDED.quiet_hours,
			metadata = EXCLUDED.metadata,
			status = 'active',
			updated_at = NOW()
		RETURNING id, user_id, channel_connection_id, scope_type, external_scope_id, mode,
			trigger_keywords, allow_memory_write, allow_tool_use, require_approval_for_outbound,
			rate_limit_per_minute, quiet_hours, status, metadata, created_at, updated_at
	`, uuid.NewString(), input.UserID, input.ChannelConnectionID, input.ScopeType, input.ExternalScopeID,
		input.Mode, input.TriggerKeywords, input.AllowMemoryWrite, input.AllowToolUse,
		input.RequireApprovalForOutbound, input.RateLimitPerMinute, input.QuietHours, input.Metadata))
}

func (s *ChannelStore) FindPolicy(ctx context.Context, connectionID, scopeType string, externalScopeID *string) (ChannelPolicy, error) {
	return scanChannelPolicy(s.db.QueryRow(ctx, `
		SELECT id, user_id, channel_connection_id, scope_type, external_scope_id, mode,
			trigger_keywords, allow_memory_write, allow_tool_use, require_approval_for_outbound,
			rate_limit_per_minute, quiet_hours, status, metadata, created_at, updated_at
		FROM channel_policies
		WHERE channel_connection_id = $1 AND scope_type = $2
			AND (($3::text IS NULL AND external_scope_id IS NULL) OR external_scope_id = $3)
			AND status = 'active'
	`, connectionID, scopeType, externalScopeID))
}

func (s *ChannelStore) FindExternalConversation(ctx context.Context, connectionID, externalID string) (ExternalConversation, error) {
	return scanExternalConversation(s.db.QueryRow(ctx, `
		SELECT id, user_id, channel_connection_id, conversation_id, external_conversation_id,
			external_conversation_type, external_title, last_message_at, status, metadata, created_at, updated_at
		FROM external_conversations
		WHERE channel_connection_id = $1 AND external_conversation_id = $2 AND status <> 'deleted'
	`, connectionID, externalID))
}

func (s *ChannelStore) CreateExternalConversation(ctx context.Context, userID, connectionID, conversationID, externalID, externalType string, externalTitle *string, metadata json.RawMessage) (ExternalConversation, error) {
	if len(metadata) == 0 {
		metadata = json.RawMessage(`{}`)
	}
	return scanExternalConversation(s.db.QueryRow(ctx, `
		INSERT INTO external_conversations (
			id, user_id, channel_connection_id, conversation_id, external_conversation_id,
			external_conversation_type, external_title, last_message_at, metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), $8)
		RETURNING id, user_id, channel_connection_id, conversation_id, external_conversation_id,
			external_conversation_type, external_title, last_message_at, status, metadata, created_at, updated_at
	`, uuid.NewString(), userID, connectionID, conversationID, externalID, externalType, externalTitle, metadata))
}

func (s *ChannelStore) ListExternalConversations(ctx context.Context, userID, connectionID string, limit int) ([]ExternalConversation, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, channel_connection_id, conversation_id, external_conversation_id,
			external_conversation_type, external_title, last_message_at, status, metadata, created_at, updated_at
		FROM external_conversations
		WHERE user_id = $1 AND channel_connection_id = $2 AND status <> 'deleted'
		ORDER BY updated_at DESC
		LIMIT $3
	`, userID, connectionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var conversations []ExternalConversation
	for rows.Next() {
		conversation, err := scanExternalConversation(rows)
		if err != nil {
			return nil, err
		}
		conversations = append(conversations, conversation)
	}
	return conversations, rows.Err()
}

func (s *ChannelStore) CreateInboxEvent(ctx context.Context, event ChannelInboxEvent) (ChannelInboxEvent, error) {
	if len(event.RawPayload) == 0 {
		event.RawPayload = json.RawMessage(`{}`)
	}
	return scanChannelInboxEvent(s.db.QueryRow(ctx, `
		INSERT INTO channel_inbox_events (
			id, user_id, channel_connection_id, external_conversation_id, conversation_id, message_id,
			event_type, external_event_id, external_sender_id, external_sender_name, raw_payload,
			normalized_text, should_trigger_agent, trigger_reason, status, processed_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (channel_connection_id, external_event_id) DO UPDATE SET
			status = channel_inbox_events.status
		RETURNING id, user_id, channel_connection_id, external_conversation_id, conversation_id, message_id,
			event_type, external_event_id, external_sender_id, external_sender_name, raw_payload,
			normalized_text, should_trigger_agent, trigger_reason, status, received_at, processed_at
	`, uuid.NewString(), event.UserID, event.ChannelConnectionID, event.ExternalConversationID, event.ConversationID,
		event.MessageID, event.EventType, event.ExternalEventID, event.ExternalSenderID, event.ExternalSenderName,
		event.RawPayload, event.NormalizedText, event.ShouldTriggerAgent, event.TriggerReason, event.Status, event.ProcessedAt))
}

func (s *ChannelStore) ListInboxEvents(ctx context.Context, userID, connectionID string, limit int) ([]ChannelInboxEvent, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, channel_connection_id, external_conversation_id, conversation_id, message_id,
			event_type, external_event_id, external_sender_id, external_sender_name, raw_payload,
			normalized_text, should_trigger_agent, trigger_reason, status, received_at, processed_at
		FROM channel_inbox_events
		WHERE user_id = $1 AND channel_connection_id = $2
		ORDER BY received_at DESC
		LIMIT $3
	`, userID, connectionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []ChannelInboxEvent
	for rows.Next() {
		event, err := scanChannelInboxEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (s *ChannelStore) CreateOutboxMessage(ctx context.Context, message ChannelOutboxMessage) (ChannelOutboxMessage, error) {
	if len(message.Payload) == 0 {
		message.Payload = json.RawMessage(`{}`)
	}
	return scanChannelOutboxMessage(s.db.QueryRow(ctx, `
		INSERT INTO channel_outbox_messages (
			id, user_id, channel_connection_id, external_conversation_id, conversation_id, agent_turn_id,
			reply_to_inbox_event_id, message_type, content, payload, requires_approval, status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, user_id, channel_connection_id, external_conversation_id, conversation_id, agent_turn_id,
			reply_to_inbox_event_id, message_type, content, payload, requires_approval, status,
			external_message_id, error_message, created_at, approved_at, sent_at
	`, uuid.NewString(), message.UserID, message.ChannelConnectionID, message.ExternalConversationID,
		message.ConversationID, message.AgentTurnID, message.ReplyToInboxEventID, message.MessageType,
		message.Content, message.Payload, message.RequiresApproval, message.Status))
}

func (s *ChannelStore) ListOutboxMessages(ctx context.Context, userID, connectionID string, status *string, limit int) ([]ChannelOutboxMessage, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, channel_connection_id, external_conversation_id, conversation_id, agent_turn_id,
			reply_to_inbox_event_id, message_type, content, payload, requires_approval, status,
			external_message_id, error_message, created_at, approved_at, sent_at
		FROM channel_outbox_messages
		WHERE user_id = $1 AND channel_connection_id = $2 AND ($3::text IS NULL OR status = $3)
		ORDER BY created_at DESC
		LIMIT $4
	`, userID, connectionID, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var messages []ChannelOutboxMessage
	for rows.Next() {
		message, err := scanChannelOutboxMessage(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	return messages, rows.Err()
}

func ToPublicChannelConnection(connection ChannelConnection) PublicChannelConnection {
	return PublicChannelConnection{
		ID:                  connection.ID,
		UserID:              connection.UserID,
		ProviderID:          connection.ProviderID,
		DisplayName:         connection.DisplayName,
		ExternalAccountID:   connection.ExternalAccountID,
		ExternalAccountName: connection.ExternalAccountName,
		HasConfig:           len(connection.EncryptedConfig) > 0 && string(connection.EncryptedConfig) != "{}",
		Status:              connection.Status,
		LastHealthStatus:    connection.LastHealthStatus,
		LastEventAt:         connection.LastEventAt,
		LastCheckedAt:       connection.LastCheckedAt,
		CreatedAt:           connection.CreatedAt,
		UpdatedAt:           connection.UpdatedAt,
	}
}

func scanChannelProvider(row pgx.Row) (ChannelProviderDefinition, error) {
	var provider ChannelProviderDefinition
	err := row.Scan(&provider.ID, &provider.Name, &provider.DisplayName, &provider.Description,
		&provider.ProviderType, &provider.AdapterType, &provider.InboundModes, &provider.OutboundModes,
		&provider.ConfigSchema, &provider.DefaultPolicy, &provider.Visibility, &provider.Status,
		&provider.Metadata, &provider.CreatedAt, &provider.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ChannelProviderDefinition{}, ErrNotFound
		}
		return ChannelProviderDefinition{}, err
	}
	return provider, nil
}

func scanChannelConnection(row pgx.Row) (ChannelConnection, error) {
	var connection ChannelConnection
	err := row.Scan(&connection.ID, &connection.UserID, &connection.ProviderID, &connection.DisplayName,
		&connection.ExternalAccountID, &connection.ExternalAccountName, &connection.EncryptedConfig,
		&connection.Status, &connection.LastHealthStatus, &connection.LastEventAt, &connection.LastCheckedAt,
		&connection.Metadata, &connection.CreatedAt, &connection.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ChannelConnection{}, ErrNotFound
		}
		return ChannelConnection{}, err
	}
	return connection, nil
}

func scanChannelPolicy(row pgx.Row) (ChannelPolicy, error) {
	var policy ChannelPolicy
	err := row.Scan(&policy.ID, &policy.UserID, &policy.ChannelConnectionID, &policy.ScopeType,
		&policy.ExternalScopeID, &policy.Mode, &policy.TriggerKeywords, &policy.AllowMemoryWrite,
		&policy.AllowToolUse, &policy.RequireApprovalForOutbound, &policy.RateLimitPerMinute,
		&policy.QuietHours, &policy.Status, &policy.Metadata, &policy.CreatedAt, &policy.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ChannelPolicy{}, ErrNotFound
		}
		return ChannelPolicy{}, err
	}
	return policy, nil
}

func scanExternalConversation(row pgx.Row) (ExternalConversation, error) {
	var conversation ExternalConversation
	err := row.Scan(&conversation.ID, &conversation.UserID, &conversation.ChannelConnectionID,
		&conversation.ConversationID, &conversation.ExternalConversationID, &conversation.ExternalConversationType,
		&conversation.ExternalTitle, &conversation.LastMessageAt, &conversation.Status, &conversation.Metadata,
		&conversation.CreatedAt, &conversation.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ExternalConversation{}, ErrNotFound
		}
		return ExternalConversation{}, err
	}
	return conversation, nil
}

func scanChannelInboxEvent(row pgx.Row) (ChannelInboxEvent, error) {
	var event ChannelInboxEvent
	err := row.Scan(&event.ID, &event.UserID, &event.ChannelConnectionID, &event.ExternalConversationID,
		&event.ConversationID, &event.MessageID, &event.EventType, &event.ExternalEventID,
		&event.ExternalSenderID, &event.ExternalSenderName, &event.RawPayload, &event.NormalizedText,
		&event.ShouldTriggerAgent, &event.TriggerReason, &event.Status, &event.ReceivedAt, &event.ProcessedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ChannelInboxEvent{}, ErrNotFound
		}
		return ChannelInboxEvent{}, err
	}
	return event, nil
}

func scanChannelOutboxMessage(row pgx.Row) (ChannelOutboxMessage, error) {
	var message ChannelOutboxMessage
	err := row.Scan(&message.ID, &message.UserID, &message.ChannelConnectionID, &message.ExternalConversationID,
		&message.ConversationID, &message.AgentTurnID, &message.ReplyToInboxEventID, &message.MessageType,
		&message.Content, &message.Payload, &message.RequiresApproval, &message.Status,
		&message.ExternalMessageID, &message.ErrorMessage, &message.CreatedAt, &message.ApprovedAt,
		&message.SentAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ChannelOutboxMessage{}, ErrNotFound
		}
		return ChannelOutboxMessage{}, err
	}
	return message, nil
}
