package store

import (
	"errors"

	"github.com/jackc/pgx/v5"
)

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
