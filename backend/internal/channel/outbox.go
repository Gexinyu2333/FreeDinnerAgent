package channel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"freedinner/backend/internal/store"
)

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

func (s *Service) SendApprovedOutboxBatch(ctx context.Context, limit int) ([]SenderRunResult, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	messages, err := s.channels.ListApprovedOutboxMessages(ctx, limit)
	if err != nil {
		return nil, err
	}
	results := make([]SenderRunResult, 0, len(messages))
	for _, message := range messages {
		_, err := s.SendOutboxMessage(ctx, message.UserID, message.ID)
		result := SenderRunResult{OutboxID: message.ID, Status: "sent"}
		if err != nil {
			errText := err.Error()
			result.Status = "failed"
			result.Error = &errText
		}
		results = append(results, result)
	}
	return results, nil
}

func (s *Service) StartSenderWorker(ctx context.Context, interval time.Duration, limit int) {
	if interval <= 0 {
		interval = 10 * time.Second
	}
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.runSenderTick(ctx, limit)
			}
		}
	}()
}

func (s *Service) runSenderTick(ctx context.Context, limit int) {
	results, err := s.SendApprovedOutboxBatch(ctx, limit)
	if err != nil {
		log.Printf("channel sender tick failed: %v", err)
		return
	}
	for _, result := range results {
		if result.Error != nil {
			log.Printf("channel sender outbox=%s status=%s error=%s", result.OutboxID, result.Status, *result.Error)
		}
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

func outboxStatus(requiresApproval bool) string {
	if requiresApproval {
		return "pending"
	}
	return "approved"
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
