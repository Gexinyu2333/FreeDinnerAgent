package channel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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
	if shouldTrigger && policy.RateLimitPerMinute > 0 {
		count, err := s.channels.CountRecentTriggeredInboxEvents(ctx, connection.UserID, connection.ID, event.ScopeType, event.ExternalScopeID, now.Add(-time.Minute))
		if err == nil && count >= policy.RateLimitPerMinute {
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

func (s *Service) ApproveOutboxMessage(ctx context.Context, userID, outboxID string) (store.ChannelOutboxMessage, error) {
	return s.channels.ResolveOutboxMessage(ctx, userID, outboxID, "approved")
}

func (s *Service) CancelOutboxMessage(ctx context.Context, userID, outboxID string) (store.ChannelOutboxMessage, error) {
	return s.channels.ResolveOutboxMessage(ctx, userID, outboxID, "cancelled")
}

func (s *Service) SendOutboxMessage(ctx context.Context, userID, outboxID string) (store.ChannelOutboxMessage, error) {
	message, err := s.channels.MarkOutboxSending(ctx, userID, outboxID)
	if err != nil {
		return store.ChannelOutboxMessage{}, err
	}
	connection, err := s.channels.FindUserConnectionByID(ctx, userID, message.ChannelConnectionID)
	if err != nil {
		return store.ChannelOutboxMessage{}, err
	}
	cfg, err := s.decryptConfig(connection.EncryptedConfig)
	if err != nil {
		_, _ = s.channels.MarkOutboxFailed(ctx, userID, outboxID, err.Error())
		return store.ChannelOutboxMessage{}, err
	}
	if strings.TrimSpace(cfg.Endpoint) == "" {
		err := errors.New("missing channel endpoint")
		_, _ = s.channels.MarkOutboxFailed(ctx, userID, outboxID, err.Error())
		return store.ChannelOutboxMessage{}, err
	}
	endpoint := strings.TrimRight(cfg.Endpoint, "/") + "/send_msg"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(message.Payload))
	if err != nil {
		_, _ = s.channels.MarkOutboxFailed(ctx, userID, outboxID, err.Error())
		return store.ChannelOutboxMessage{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(cfg.AccessToken) != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.AccessToken)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		_, _ = s.channels.MarkOutboxFailed(ctx, userID, outboxID, err.Error())
		return store.ChannelOutboxMessage{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err := fmt.Errorf("channel send status %d: %s", resp.StatusCode, string(body))
		_, _ = s.channels.MarkOutboxFailed(ctx, userID, outboxID, err.Error())
		return store.ChannelOutboxMessage{}, err
	}
	externalID := extractExternalMessageID(body)
	return s.channels.MarkOutboxSent(ctx, userID, outboxID, externalID)
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
	return shouldTriggerAt(event, policy, botQQ, time.Now())
}

func shouldTriggerAt(event normalizedEvent, policy store.ChannelPolicy, botQQ *string, now time.Time) (bool, string) {
	if inQuietHours(policy.QuietHours, now) {
		return false, "quiet_hours"
	}
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

func buildOutboxPayload(adapter ChannelAdapter, connection store.ChannelConnection, external store.ExternalConversation, content string) json.RawMessage {
	payload, err := adapter.BuildSendPayload(context.Background(), OutboxMessage{
		Connection:           connection,
		ExternalConversation: external,
		Message: store.ChannelOutboxMessage{
			Content: content,
		},
	})
	if err != nil || len(payload) == 0 {
		return json.RawMessage(`{}`)
	}
	return payload
}

func buildOneBotSendPayload(external store.ExternalConversation, content string) (json.RawMessage, error) {
	payload := map[string]any{
		"message": content,
	}
	switch external.ExternalConversationType {
	case "group_chat":
		payload["message_type"] = "group"
		payload["group_id"] = external.ExternalConversationID
	default:
		payload["message_type"] = "private"
		payload["user_id"] = external.ExternalConversationID
	}
	return json.Marshal(payload)
}

type quietHoursConfig struct {
	Enabled  bool   `json:"enabled"`
	Timezone string `json:"timezone"`
	Start    string `json:"start"`
	End      string `json:"end"`
}

func inQuietHours(raw json.RawMessage, now time.Time) bool {
	if len(raw) == 0 || string(raw) == "{}" || string(raw) == "null" {
		return false
	}
	var cfg quietHoursConfig
	if err := json.Unmarshal(raw, &cfg); err != nil || !cfg.Enabled {
		return false
	}
	location := time.Local
	if strings.TrimSpace(cfg.Timezone) != "" {
		if loaded, err := time.LoadLocation(strings.TrimSpace(cfg.Timezone)); err == nil {
			location = loaded
		}
	}
	localNow := now.In(location)
	startMinute, ok := parseClockMinute(cfg.Start)
	if !ok {
		return false
	}
	endMinute, ok := parseClockMinute(cfg.End)
	if !ok {
		return false
	}
	currentMinute := localNow.Hour()*60 + localNow.Minute()
	if startMinute == endMinute {
		return true
	}
	if startMinute < endMinute {
		return currentMinute >= startMinute && currentMinute < endMinute
	}
	return currentMinute >= startMinute || currentMinute < endMinute
}

func parseClockMinute(value string) (int, bool) {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) < 2 {
		return 0, false
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return 0, false
	}
	minute, err := strconv.Atoi(parts[1])
	if err != nil || minute < 0 || minute > 59 {
		return 0, false
	}
	return hour*60 + minute, true
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

func extractExternalMessageID(body []byte) *string {
	var decoded struct {
		Data struct {
			MessageID any `json:"message_id"`
		} `json:"data"`
		MessageID any `json:"message_id"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil
	}
	value := valueToString(decoded.MessageID)
	if value == "" {
		value = valueToString(decoded.Data.MessageID)
	}
	if value == "" {
		return nil
	}
	return &value
}
