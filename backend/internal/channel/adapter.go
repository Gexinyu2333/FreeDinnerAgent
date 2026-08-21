package channel

import (
	"context"
	"encoding/json"

	"freedinner/backend/internal/store"
)

type InboundRequest struct {
	Connection     store.ChannelConnection
	ProvidedSecret string
	RawPayload     []byte
}

type OutboxMessage struct {
	Connection           store.ChannelConnection
	ExternalConversation store.ExternalConversation
	Message              store.ChannelOutboxMessage
}

type SendResult struct {
	ExternalMessageID *string `json:"external_message_id,omitempty"`
	Status            string  `json:"status"`
	RawResponse       any     `json:"raw_response,omitempty"`
}

type HealthStatus struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type ChannelAdapter interface {
	VerifyInbound(ctx context.Context, req InboundRequest, cfg connectionConfig) error
	NormalizeEvent(ctx context.Context, raw []byte, botAccountID *string) (normalizedEvent, error)
	BuildSendPayload(ctx context.Context, msg OutboxMessage) (json.RawMessage, error)
	HealthCheck(ctx context.Context, conn store.ChannelConnection, cfg connectionConfig) (HealthStatus, error)
}

type OneBotAdapter struct{}

func (OneBotAdapter) VerifyInbound(ctx context.Context, req InboundRequest, cfg connectionConfig) error {
	if cfg.WebhookSecret != "" && req.ProvidedSecret != cfg.WebhookSecret {
		return ErrInvalidWebhookSecret
	}
	return nil
}

func (OneBotAdapter) NormalizeEvent(ctx context.Context, raw []byte, botAccountID *string) (normalizedEvent, error) {
	return normalizeOneBot(raw, botAccountID)
}

func (OneBotAdapter) BuildSendPayload(ctx context.Context, msg OutboxMessage) (json.RawMessage, error) {
	return buildOneBotSendPayload(msg.ExternalConversation, msg.Message.Content)
}

func (OneBotAdapter) HealthCheck(ctx context.Context, conn store.ChannelConnection, cfg connectionConfig) (HealthStatus, error) {
	if cfg.Endpoint == "" {
		return HealthStatus{Status: "unhealthy", Message: "missing endpoint"}, nil
	}
	return HealthStatus{Status: "unknown", Message: "network health check is not enabled in MVP"}, nil
}
