package store

import (
	"encoding/json"
	"time"

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
	ID                  string                            `json:"id"`
	UserID              string                            `json:"user_id"`
	ProviderID          string                            `json:"provider_id"`
	DisplayName         string                            `json:"display_name"`
	ExternalAccountID   *string                           `json:"external_account_id"`
	ExternalAccountName *string                           `json:"external_account_name"`
	Endpoints           []PublicChannelConnectionEndpoint `json:"endpoints"`
	HasConfig           bool                              `json:"has_config"`
	Status              string                            `json:"status"`
	LastHealthStatus    *string                           `json:"last_health_status"`
	LastEventAt         *time.Time                        `json:"last_event_at"`
	LastCheckedAt       *time.Time                        `json:"last_checked_at"`
	CreatedAt           time.Time                         `json:"created_at"`
	UpdatedAt           time.Time                         `json:"updated_at"`
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

type ChannelConnectionEndpoint struct {
	ID                  string          `json:"id"`
	UserID              string          `json:"user_id"`
	ChannelConnectionID string          `json:"channel_connection_id"`
	EndpointType        string          `json:"endpoint_type"`
	DisplayName         string          `json:"display_name"`
	Direction           string          `json:"direction"`
	Transport           string          `json:"transport"`
	URL                 string          `json:"url"`
	EncryptedConfig     json.RawMessage `json:"-"`
	Status              string          `json:"status"`
	Metadata            json.RawMessage `json:"metadata"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

type PublicChannelConnectionEndpoint struct {
	ID                  string          `json:"id"`
	ChannelConnectionID string          `json:"channel_connection_id"`
	EndpointType        string          `json:"endpoint_type"`
	DisplayName         string          `json:"display_name"`
	Direction           string          `json:"direction"`
	Transport           string          `json:"transport"`
	URL                 string          `json:"url"`
	HasSecret           bool            `json:"has_secret"`
	Status              string          `json:"status"`
	Metadata            json.RawMessage `json:"metadata"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

type ChannelConnectionEndpointCreate struct {
	UserID              string
	ChannelConnectionID string
	EndpointType        string
	DisplayName         string
	Direction           string
	Transport           string
	URL                 string
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

func ToPublicChannelConnectionEndpoint(endpoint ChannelConnectionEndpoint) PublicChannelConnectionEndpoint {
	return PublicChannelConnectionEndpoint{
		ID:                  endpoint.ID,
		ChannelConnectionID: endpoint.ChannelConnectionID,
		EndpointType:        endpoint.EndpointType,
		DisplayName:         endpoint.DisplayName,
		Direction:           endpoint.Direction,
		Transport:           endpoint.Transport,
		URL:                 endpoint.URL,
		HasSecret:           len(endpoint.EncryptedConfig) > 0 && string(endpoint.EncryptedConfig) != "{}",
		Status:              endpoint.Status,
		Metadata:            endpoint.Metadata,
		CreatedAt:           endpoint.CreatedAt,
		UpdatedAt:           endpoint.UpdatedAt,
	}
}
