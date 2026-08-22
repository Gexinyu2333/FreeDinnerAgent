package channel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"freedinner/backend/internal/store"
)

func (s *Service) HandleWebhook(ctx context.Context, connectionID, providedSecret string, rawPayload []byte) (WebhookResult, error) {
	connection, err := s.channels.FindConnectionByID(ctx, connectionID)
	if err != nil {
		return WebhookResult{}, err
	}
	cfg, err := s.decryptConfig(connection.EncryptedConfig)
	if err != nil {
		return WebhookResult{}, err
	}
	adapter := s.adapterForConnection(ctx, connection)
	if err := adapter.VerifyInbound(ctx, InboundRequest{
		Connection:     connection,
		ProvidedSecret: providedSecret,
		RawPayload:     rawPayload,
	}, cfg); err != nil {
		return WebhookResult{}, err
	}

	event, err := adapter.NormalizeEvent(ctx, rawPayload, connection.ExternalAccountID)
	if err != nil {
		return WebhookResult{}, err
	}
	policy := s.resolvePolicy(ctx, connection, event)
	now := time.Now()
	shouldTrigger, reason := shouldTriggerAt(event, policy, connection.ExternalAccountID, now)
	if shouldTrigger {
		blocked, err := s.isRateLimited(ctx, connection, event, policy, now)
		if err == nil && blocked {
			shouldTrigger = false
			reason = "rate_limited"
		}
	}
	status := "ignored"
	if shouldTrigger {
		status = "processed"
	}

	var externalConversation *store.ExternalConversation
	var conversation *store.Conversation
	var message *store.Message
	var outbox *store.ChannelOutboxMessage

	if shouldTrigger {
		external, conv, msg, err := s.ensureConversationAndMessage(ctx, connection, event)
		if err != nil {
			return WebhookResult{}, err
		}
		externalConversation = &external
		conversation = &conv
		message = &msg
	}

	var externalConversationID *string
	var conversationID *string
	var messageID *string
	if externalConversation != nil {
		externalConversationID = &externalConversation.ID
	}
	if conversation != nil {
		conversationID = &conversation.ID
	}
	if message != nil {
		messageID = &message.ID
	}
	processedAt := time.Now()
	inbox, err := s.channels.CreateInboxEvent(ctx, store.ChannelInboxEvent{
		UserID:                 connection.UserID,
		ChannelConnectionID:    connection.ID,
		ExternalConversationID: externalConversationID,
		ConversationID:         conversationID,
		MessageID:              messageID,
		EventType:              event.EventType,
		ExternalEventID:        event.ExternalEventID,
		ExternalSenderID:       event.ExternalSenderID,
		ExternalSenderName:     event.ExternalSenderName,
		RawPayload:             event.RawPayload,
		NormalizedText:         &event.Text,
		ShouldTriggerAgent:     shouldTrigger,
		TriggerReason:          &reason,
		Status:                 status,
		ProcessedAt:            &processedAt,
	})
	if err != nil {
		return WebhookResult{}, err
	}

	if shouldTrigger && conversation != nil && externalConversation != nil {
		content := fmt.Sprintf("已收到 QQ 消息：%s\n\n当前 Agent 暂时无法生成回复，请稍后重试。", event.Text)
		var agentTurnID *string
		if s.responder != nil && message != nil {
			result, err := s.responder.RespondToExistingMessage(ctx, connection.UserID, conversation.ID, *message)
			if err == nil && strings.TrimSpace(result.AssistantMessage.Content) != "" {
				content = result.AssistantMessage.Content
				agentTurnID = &result.TurnID
				message = &result.AssistantMessage
			} else {
				metadata, _ := json.Marshal(map[string]any{"source": "channel_adapter_fallback", "inbox_event_id": inbox.ID})
				assistantMessage, createErr := s.conversations.CreateAssistantMessage(ctx, connection.UserID, conversation.ID, content, metadata)
				if createErr != nil {
					return WebhookResult{}, createErr
				}
				message = &assistantMessage
			}
		} else {
			metadata, _ := json.Marshal(map[string]any{"source": "channel_adapter_fallback", "inbox_event_id": inbox.ID})
			assistantMessage, err := s.conversations.CreateAssistantMessage(ctx, connection.UserID, conversation.ID, content, metadata)
			if err != nil {
				return WebhookResult{}, err
			}
			message = &assistantMessage
		}
		outboxMessage, err := s.channels.CreateOutboxMessage(ctx, store.ChannelOutboxMessage{
			UserID:                 connection.UserID,
			ChannelConnectionID:    connection.ID,
			ExternalConversationID: &externalConversation.ID,
			ConversationID:         &conversation.ID,
			AgentTurnID:            agentTurnID,
			ReplyToInboxEventID:    &inbox.ID,
			MessageType:            "text",
			Content:                content,
			Payload:                buildOutboxPayload(adapter, connection, *externalConversation, content),
			RequiresApproval:       policy.RequireApprovalForOutbound,
			Status:                 outboxStatus(policy.RequireApprovalForOutbound),
		})
		if err != nil {
			return WebhookResult{}, err
		}
		outbox = &outboxMessage
	}

	return WebhookResult{
		InboxEvent:           inbox,
		ExternalConversation: externalConversation,
		Conversation:         conversation,
		Message:              message,
		OutboxMessage:        outbox,
	}, nil
}

func (s *Service) ensureConversationAndMessage(ctx context.Context, connection store.ChannelConnection, event normalizedEvent) (store.ExternalConversation, store.Conversation, store.Message, error) {
	external, err := s.channels.FindExternalConversation(ctx, connection.ID, event.ExternalConversationID)
	if err == nil {
		msg, err := s.conversations.CreateUserMessage(ctx, connection.UserID, external.ConversationID, event.Text)
		if err != nil {
			return store.ExternalConversation{}, store.Conversation{}, store.Message{}, err
		}
		conv, err := s.conversations.FindByID(ctx, connection.UserID, external.ConversationID)
		return external, conv, msg, err
	}
	if !errors.Is(err, store.ErrNotFound) {
		return store.ExternalConversation{}, store.Conversation{}, store.Message{}, err
	}

	title := "QQ " + event.ExternalConversationID
	if event.ExternalTitle != nil && *event.ExternalTitle != "" {
		title = "QQ " + *event.ExternalTitle
	}
	conversation, err := s.conversations.CreateWithChannel(ctx, connection.UserID, title, "qq")
	if err != nil {
		return store.ExternalConversation{}, store.Conversation{}, store.Message{}, err
	}
	external, err = s.channels.CreateExternalConversation(ctx, connection.UserID, connection.ID, conversation.ID,
		event.ExternalConversationID, event.ExternalConversationType, event.ExternalTitle, nil)
	if err != nil {
		return store.ExternalConversation{}, store.Conversation{}, store.Message{}, err
	}
	message, err := s.conversations.CreateUserMessage(ctx, connection.UserID, conversation.ID, event.Text)
	return external, conversation, message, err
}
