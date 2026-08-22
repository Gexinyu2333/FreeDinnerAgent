package channel

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"freedinner/backend/internal/secret"
	"freedinner/backend/internal/store"
)

var ErrInvalidWebhookSecret = errors.New("invalid webhook secret")

type Service struct {
	channels      *store.ChannelStore
	conversations *store.ConversationStore
	responder     AgentResponder
	crypto        secret.Crypto
	adapters      map[string]ChannelAdapter
	httpClient    *http.Client
}

type AgentResponder interface {
	RespondToExistingMessage(ctx context.Context, userID, conversationID string, message store.Message) (store.SendMessageResult, error)
}

type CreateConnectionInput struct {
	UserID              string
	ProviderID          string
	DisplayName         string
	ExternalAccountID   *string
	ExternalAccountName *string
	Config              json.RawMessage
}

type UpsertPolicyInput struct {
	UserID                     string
	ConnectionID               string
	ScopeType                  string
	ExternalScopeID            *string
	Mode                       string
	TriggerKeywords            []string
	AllowMemoryWrite           bool
	AllowToolUse               bool
	RequireApprovalForOutbound bool
	RateLimitPerMinute         int
	RateLimitPolicy            json.RawMessage
}

type WebhookResult struct {
	InboxEvent           store.ChannelInboxEvent     `json:"inbox_event"`
	ExternalConversation *store.ExternalConversation `json:"external_conversation,omitempty"`
	Conversation         *store.Conversation         `json:"conversation,omitempty"`
	Message              *store.Message              `json:"message,omitempty"`
	OutboxMessage        *store.ChannelOutboxMessage `json:"outbox_message,omitempty"`
}

type SenderRunResult struct {
	OutboxID string  `json:"outbox_id"`
	Status   string  `json:"status"`
	Error    *string `json:"error,omitempty"`
}

type normalizedEvent struct {
	EventType                string
	ExternalEventID          *string
	ExternalConversationID   string
	ExternalConversationType string
	ExternalTitle            *string
	ExternalSenderID         *string
	ExternalSenderName       *string
	Text                     string
	ScopeType                string
	ExternalScopeID          *string
	RawPayload               json.RawMessage
}

func NewService(channels *store.ChannelStore, conversations *store.ConversationStore, crypto secret.Crypto, responder AgentResponder) *Service {
	return &Service{
		channels:      channels,
		conversations: conversations,
		responder:     responder,
		crypto:        crypto,
		adapters: map[string]ChannelAdapter{
			"qq": OneBotAdapter{},
		},
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (s *Service) EnsureBuiltins(ctx context.Context) error {
	return s.channels.EnsureBuiltinProviders(ctx)
}

func (s *Service) ListProviders(ctx context.Context) ([]store.ChannelProviderDefinition, error) {
	return s.channels.ListProviders(ctx)
}

func (s *Service) CreateConnection(ctx context.Context, input CreateConnectionInput) (store.ChannelConnection, error) {
	encryptedConfig, err := s.encryptConfig(input.Config)
	if err != nil {
		return store.ChannelConnection{}, err
	}
	connection, err := s.channels.CreateConnection(ctx, store.ChannelConnectionCreate{
		UserID:              input.UserID,
		ProviderID:          input.ProviderID,
		DisplayName:         strings.TrimSpace(input.DisplayName),
		ExternalAccountID:   trimOptional(input.ExternalAccountID),
		ExternalAccountName: trimOptional(input.ExternalAccountName),
		EncryptedConfig:     encryptedConfig,
	})
	if err != nil {
		return store.ChannelConnection{}, err
	}

	_, _ = s.channels.UpsertPolicy(ctx, store.ChannelPolicyUpsert{
		UserID:                     input.UserID,
		ChannelConnectionID:        connection.ID,
		ScopeType:                  "private_chat",
		Mode:                       "auto_reply",
		AllowMemoryWrite:           true,
		AllowToolUse:               true,
		RequireApprovalForOutbound: false,
		RateLimitPerMinute:         6,
	})
	_, _ = s.channels.UpsertPolicy(ctx, store.ChannelPolicyUpsert{
		UserID:                     input.UserID,
		ChannelConnectionID:        connection.ID,
		ScopeType:                  "group_chat",
		Mode:                       "mention_only",
		AllowMemoryWrite:           true,
		AllowToolUse:               true,
		RequireApprovalForOutbound: true,
		RateLimitPerMinute:         6,
	})
	return connection, nil
}

func (s *Service) ListConnections(ctx context.Context, userID string) ([]store.ChannelConnection, error) {
	return s.channels.ListConnections(ctx, userID)
}

func (s *Service) UpsertPolicy(ctx context.Context, input UpsertPolicyInput) (store.ChannelPolicy, error) {
	if _, err := s.channels.FindUserConnectionByID(ctx, input.UserID, input.ConnectionID); err != nil {
		return store.ChannelPolicy{}, err
	}
	return s.channels.UpsertPolicy(ctx, store.ChannelPolicyUpsert{
		UserID:                     input.UserID,
		ChannelConnectionID:        input.ConnectionID,
		ScopeType:                  normalizeScopeType(input.ScopeType),
		ExternalScopeID:            trimOptional(input.ExternalScopeID),
		Mode:                       normalizeMode(input.Mode),
		TriggerKeywords:            input.TriggerKeywords,
		AllowMemoryWrite:           input.AllowMemoryWrite,
		AllowToolUse:               input.AllowToolUse,
		RequireApprovalForOutbound: input.RequireApprovalForOutbound,
		RateLimitPerMinute:         input.RateLimitPerMinute,
		Metadata:                   rateLimitMetadata(input.RateLimitPolicy),
	})
}

func (s *Service) adapterForConnection(ctx context.Context, connection store.ChannelConnection) ChannelAdapter {
	providers, err := s.channels.ListProviders(ctx)
	if err == nil {
		for _, provider := range providers {
			if provider.ID == connection.ProviderID {
				if adapter, ok := s.adapters[provider.ProviderType]; ok {
					return adapter
				}
			}
		}
	}
	return OneBotAdapter{}
}
