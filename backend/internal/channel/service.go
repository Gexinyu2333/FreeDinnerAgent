package channel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"freedinner/backend/internal/secret"
	"freedinner/backend/internal/store"
)

var ErrInvalidWebhookSecret = errors.New("invalid webhook secret")

type Service struct {
	channels      *store.ChannelStore
	conversations *store.ConversationStore
	crypto        secret.Crypto
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
}

type WebhookResult struct {
	InboxEvent           store.ChannelInboxEvent     `json:"inbox_event"`
	ExternalConversation *store.ExternalConversation `json:"external_conversation,omitempty"`
	Conversation         *store.Conversation         `json:"conversation,omitempty"`
	Message              *store.Message              `json:"message,omitempty"`
	OutboxMessage        *store.ChannelOutboxMessage `json:"outbox_message,omitempty"`
}

type connectionConfig struct {
	Endpoint      string `json:"endpoint"`
	AccessToken   string `json:"access_token"`
	WebhookSecret string `json:"webhook_secret"`
}

type oneBotEvent struct {
	PostType    string          `json:"post_type"`
	MessageType string          `json:"message_type"`
	MessageID   any             `json:"message_id"`
	UserID      any             `json:"user_id"`
	GroupID     any             `json:"group_id"`
	RawMessage  string          `json:"raw_message"`
	Message     json.RawMessage `json:"message"`
	Sender      struct {
		Nickname string `json:"nickname"`
		Card     string `json:"card"`
	} `json:"sender"`
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

func NewService(channels *store.ChannelStore, conversations *store.ConversationStore, crypto secret.Crypto) *Service {
	return &Service{channels: channels, conversations: conversations, crypto: crypto}
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
	})
}

func (s *Service) HandleWebhook(ctx context.Context, connectionID, providedSecret string, rawPayload []byte) (WebhookResult, error) {
	connection, err := s.channels.FindConnectionByID(ctx, connectionID)
	if err != nil {
		return WebhookResult{}, err
	}
	cfg, err := s.decryptConfig(connection.EncryptedConfig)
	if err != nil {
		return WebhookResult{}, err
	}
	if cfg.WebhookSecret != "" && providedSecret != cfg.WebhookSecret {
		return WebhookResult{}, ErrInvalidWebhookSecret
	}

	event, err := normalizeOneBot(rawPayload, connection.ExternalAccountID)
	if err != nil {
		return WebhookResult{}, err
	}
	policy := s.resolvePolicy(ctx, connection, event)
	shouldTrigger, reason := shouldTrigger(event, policy, connection.ExternalAccountID)
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
		content := fmt.Sprintf("已收到 QQ 消息：%s\n\n当前 Step 10 MVP 已完成渠道入站、会话映射和 outbox 草稿；后续会把这里接入完整 Agent Loop 生成真实回复。", event.Text)
		metadata, _ := json.Marshal(map[string]any{"source": "channel_adapter", "inbox_event_id": inbox.ID})
		assistantMessage, err := s.conversations.CreateAssistantMessage(ctx, connection.UserID, conversation.ID, content, metadata)
		if err != nil {
			return WebhookResult{}, err
		}
		message = &assistantMessage
		outboxMessage, err := s.channels.CreateOutboxMessage(ctx, store.ChannelOutboxMessage{
			UserID:                 connection.UserID,
			ChannelConnectionID:    connection.ID,
			ExternalConversationID: &externalConversation.ID,
			ConversationID:         &conversation.ID,
			ReplyToInboxEventID:    &inbox.ID,
			MessageType:            "text",
			Content:                content,
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

func (s *Service) ListExternalConversations(ctx context.Context, userID, connectionID string, limit int) ([]store.ExternalConversation, error) {
	if _, err := s.channels.FindUserConnectionByID(ctx, userID, connectionID); err != nil {
		return nil, err
	}
	return s.channels.ListExternalConversations(ctx, userID, connectionID, limit)
}

func (s *Service) ListInboxEvents(ctx context.Context, userID, connectionID string, limit int) ([]store.ChannelInboxEvent, error) {
	if _, err := s.channels.FindUserConnectionByID(ctx, userID, connectionID); err != nil {
		return nil, err
	}
	return s.channels.ListInboxEvents(ctx, userID, connectionID, limit)
}

func (s *Service) ListOutboxMessages(ctx context.Context, userID, connectionID string, status *string, limit int) ([]store.ChannelOutboxMessage, error) {
	if _, err := s.channels.FindUserConnectionByID(ctx, userID, connectionID); err != nil {
		return nil, err
	}
	return s.channels.ListOutboxMessages(ctx, userID, connectionID, status, limit)
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

func (s *Service) resolvePolicy(ctx context.Context, connection store.ChannelConnection, event normalizedEvent) store.ChannelPolicy {
	policy, err := s.channels.FindPolicy(ctx, connection.ID, event.ScopeType, event.ExternalScopeID)
	if err == nil {
		return policy
	}
	policy, err = s.channels.FindPolicy(ctx, connection.ID, event.ScopeType, nil)
	if err == nil {
		return policy
	}
	return store.ChannelPolicy{
		ScopeType:                  event.ScopeType,
		Mode:                       defaultMode(event.ScopeType),
		AllowMemoryWrite:           true,
		AllowToolUse:               true,
		RequireApprovalForOutbound: event.ScopeType == "group_chat",
		RateLimitPerMinute:         6,
	}
}

func normalizeOneBot(rawPayload []byte, botQQ *string) (normalizedEvent, error) {
	var raw oneBotEvent
	if err := json.Unmarshal(rawPayload, &raw); err != nil {
		return normalizedEvent{}, err
	}
	payload := json.RawMessage(rawPayload)
	if raw.PostType != "message" {
		eventID := eventID(raw.MessageID)
		return normalizedEvent{EventType: "system", ExternalEventID: eventID, Text: raw.PostType, RawPayload: payload}, nil
	}

	text := strings.TrimSpace(raw.RawMessage)
	if text == "" {
		text = strings.TrimSpace(string(raw.Message))
	}
	senderID := valueToString(raw.UserID)
	senderName := strings.TrimSpace(raw.Sender.Card)
	if senderName == "" {
		senderName = strings.TrimSpace(raw.Sender.Nickname)
	}
	eventID := eventID(raw.MessageID)
	senderNamePtr := stringPtrOrNil(senderName)
	senderIDPtr := stringPtrOrNil(senderID)

	if raw.MessageType == "group" {
		groupID := valueToString(raw.GroupID)
		title := "群聊 " + groupID
		return normalizedEvent{
			EventType:                "message_created",
			ExternalEventID:          eventID,
			ExternalConversationID:   groupID,
			ExternalConversationType: "group_chat",
			ExternalTitle:            &title,
			ExternalSenderID:         senderIDPtr,
			ExternalSenderName:       senderNamePtr,
			Text:                     stripBotMention(text, botQQ),
			ScopeType:                "group_chat",
			ExternalScopeID:          &groupID,
			RawPayload:               payload,
		}, nil
	}

	title := "私聊 " + senderID
	return normalizedEvent{
		EventType:                "message_created",
		ExternalEventID:          eventID,
		ExternalConversationID:   senderID,
		ExternalConversationType: "private_chat",
		ExternalTitle:            &title,
		ExternalSenderID:         senderIDPtr,
		ExternalSenderName:       senderNamePtr,
		Text:                     text,
		ScopeType:                "private_chat",
		ExternalScopeID:          &senderID,
		RawPayload:               payload,
	}, nil
}

func shouldTrigger(event normalizedEvent, policy store.ChannelPolicy, botQQ *string) (bool, string) {
	switch policy.Mode {
	case "disabled", "silent_listen":
		return false, policy.Mode
	case "auto_reply":
		return true, "auto_reply"
	case "keyword":
		for _, keyword := range policy.TriggerKeywords {
			if keyword != "" && strings.Contains(event.Text, keyword) {
				return true, "keyword"
			}
		}
		return false, "keyword_missed"
	default:
		if event.ScopeType == "group_chat" {
			if botQQ != nil && strings.Contains(string(event.RawPayload), "[CQ:at,qq="+*botQQ+"]") {
				return true, "mention"
			}
			return false, "mention_required"
		}
		return true, "private_chat"
	}
}

func (s *Service) encryptConfig(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 {
		raw = json.RawMessage(`{}`)
	}
	ciphertext, err := s.crypto.Encrypt(string(raw))
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]string{"ciphertext": ciphertext})
}

func (s *Service) decryptConfig(raw json.RawMessage) (connectionConfig, error) {
	var wrapper struct {
		Ciphertext string `json:"ciphertext"`
	}
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		return connectionConfig{}, err
	}
	if wrapper.Ciphertext == "" {
		return connectionConfig{}, nil
	}
	plaintext, err := s.crypto.Decrypt(wrapper.Ciphertext)
	if err != nil {
		return connectionConfig{}, err
	}
	var cfg connectionConfig
	if err := json.Unmarshal([]byte(plaintext), &cfg); err != nil {
		return connectionConfig{}, err
	}
	return cfg, nil
}

func outboxStatus(requiresApproval bool) string {
	if requiresApproval {
		return "pending"
	}
	return "approved"
}

func defaultMode(scopeType string) string {
	if scopeType == "group_chat" {
		return "mention_only"
	}
	return "auto_reply"
}

func normalizeScopeType(value string) string {
	switch strings.TrimSpace(value) {
	case "group_chat", "all":
		return strings.TrimSpace(value)
	default:
		return "private_chat"
	}
}

func normalizeMode(value string) string {
	switch strings.TrimSpace(value) {
	case "disabled", "silent_listen", "mention_only", "keyword", "auto_reply":
		return strings.TrimSpace(value)
	default:
		return "mention_only"
	}
}

func trimOptional(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func eventID(value any) *string {
	stringValue := valueToString(value)
	if stringValue == "" {
		return nil
	}
	return &stringValue
}

func valueToString(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case int:
		return strconv.Itoa(v)
	default:
		return ""
	}
}

func stringPtrOrNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func stripBotMention(text string, botQQ *string) string {
	if botQQ == nil || *botQQ == "" {
		return strings.TrimSpace(text)
	}
	return strings.TrimSpace(strings.ReplaceAll(text, "[CQ:at,qq="+*botQQ+"]", ""))
}
