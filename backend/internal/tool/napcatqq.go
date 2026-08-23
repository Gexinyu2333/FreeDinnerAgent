package tool

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"freedinner/backend/internal/store"
)

type napcatEndpointConfig struct {
	AccessToken string `json:"access_token"`
}

func (s *Service) sendNapCatText(ctx context.Context, input ExecuteInput) (any, error) {
	var args struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(input.Arguments, &args); err != nil {
		return nil, err
	}
	text := strings.TrimSpace(args.Text)
	if text == "" {
		return nil, errors.New("text is required")
	}
	payload, err := s.currentConversationSendPayload(ctx, input.UserID, input.ConversationID, text)
	if err != nil {
		return nil, err
	}
	return s.callNapCat(ctx, input.UserID, input.ConversationID, "send_msg", payload)
}

func (s *Service) sendNapCatPicture(ctx context.Context, input ExecuteInput) (any, error) {
	var args struct {
		FileName string `json:"file_name"`
		Caption  string `json:"caption"`
	}
	if err := json.Unmarshal(input.Arguments, &args); err != nil {
		return nil, err
	}
	fileName := strings.TrimSpace(args.FileName)
	if fileName == "" {
		return nil, errors.New("file_name is required")
	}
	image, err := loadBuiltinPicture(fileName)
	if err != nil {
		return nil, err
	}
	message := make([]map[string]any, 0, 2)
	if caption := strings.TrimSpace(args.Caption); caption != "" {
		message = append(message, map[string]any{
			"type": "text",
			"data": map[string]any{"text": caption},
		})
	}
	message = append(message, map[string]any{
		"type": "image",
		"data": map[string]any{"file": "base64://" + image},
	})
	payload, err := s.currentConversationSendPayload(ctx, input.UserID, input.ConversationID, message)
	if err != nil {
		return nil, err
	}
	result, err := s.callNapCat(ctx, input.UserID, input.ConversationID, "send_msg", payload)
	if err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "file_name": fileName, "response": result}, nil
}

func (s *Service) pokeNapCatUser(ctx context.Context, input ExecuteInput) (any, error) {
	var args struct {
		UserID string `json:"user_id"`
	}
	_ = json.Unmarshal(input.Arguments, &args)
	conversation, err := s.currentNapCatConversation(ctx, input.UserID, input.ConversationID)
	if err != nil {
		return nil, err
	}
	targetUserID := strings.TrimSpace(args.UserID)
	if targetUserID == "" && conversation.ExternalConversationType != nil && *conversation.ExternalConversationType == "private_chat" && conversation.ExternalConversationID != nil {
		targetUserID = *conversation.ExternalConversationID
	}
	if targetUserID == "" {
		return nil, errors.New("user_id is required when current message sender cannot be inferred")
	}
	payload := map[string]any{"user_id": targetUserID}
	if conversation.ExternalConversationType != nil && *conversation.ExternalConversationType == "group_chat" && conversation.ExternalConversationID != nil {
		payload["group_id"] = *conversation.ExternalConversationID
	}
	return s.callNapCat(ctx, input.UserID, input.ConversationID, "send_poke", payload)
}

func (s *Service) currentConversationSendPayload(ctx context.Context, userID, conversationID string, message any) (map[string]any, error) {
	conversation, err := s.currentNapCatConversation(ctx, userID, conversationID)
	if err != nil {
		return nil, err
	}
	if conversation.ExternalConversationID == nil || strings.TrimSpace(*conversation.ExternalConversationID) == "" {
		return nil, errors.New("current channel conversation is missing external conversation id")
	}
	payload := map[string]any{"message": message}
	if conversation.ExternalConversationType != nil && *conversation.ExternalConversationType == "group_chat" {
		payload["message_type"] = "group"
		payload["group_id"] = *conversation.ExternalConversationID
	} else {
		payload["message_type"] = "private"
		payload["user_id"] = *conversation.ExternalConversationID
	}
	return payload, nil
}

func (s *Service) currentNapCatConversation(ctx context.Context, userID, conversationID string) (store.Conversation, error) {
	conversation := s.napcatChannelContext(ctx, userID, conversationID)
	if conversation == nil {
		return store.Conversation{}, errors.New("napcatqq tools are only available inside a channel conversation")
	}
	return *conversation, nil
}

func (s *Service) callNapCat(ctx context.Context, userID, conversationID, action string, payload any) (map[string]any, error) {
	conversation, err := s.currentNapCatConversation(ctx, userID, conversationID)
	if err != nil {
		return nil, err
	}
	if s.channels == nil {
		return nil, errors.New("channel store is not configured")
	}
	endpoint, err := s.channels.FindEndpointByType(ctx, userID, *conversation.ChannelConnectionID, "message_api")
	if err != nil {
		return nil, errors.New("missing channel message_api endpoint")
	}
	if strings.TrimSpace(endpoint.URL) == "" {
		return nil, errors.New("channel message_api endpoint url is empty")
	}
	cfg, err := s.decryptNapCatEndpointConfig(endpoint.EncryptedConfig)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(endpoint.URL, "/")+"/"+action, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token := strings.TrimSpace(cfg.AccessToken); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	responseBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("napcat action %s status %d: %s", action, resp.StatusCode, string(responseBody))
	}
	var decoded map[string]any
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		decoded = map[string]any{"raw": string(responseBody)}
	}
	return map[string]any{"ok": true, "action": action, "response": decoded}, nil
}

func (s *Service) decryptNapCatEndpointConfig(raw json.RawMessage) (napcatEndpointConfig, error) {
	var wrapper struct {
		Ciphertext string `json:"ciphertext"`
	}
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		return napcatEndpointConfig{}, err
	}
	if wrapper.Ciphertext == "" {
		return napcatEndpointConfig{}, nil
	}
	plaintext, err := s.crypto.Decrypt(wrapper.Ciphertext)
	if err != nil {
		return napcatEndpointConfig{}, err
	}
	var cfg napcatEndpointConfig
	if err := json.Unmarshal([]byte(plaintext), &cfg); err != nil {
		return napcatEndpointConfig{}, err
	}
	return cfg, nil
}

func loadBuiltinPicture(fileName string) (string, error) {
	clean := filepath.Base(fileName)
	if clean != fileName || strings.HasPrefix(clean, ".") {
		return "", errors.New("file_name must be a file in backend/internal/tool/picture")
	}
	candidates := []string{
		filepath.Join("internal", "tool", "picture", clean),
		filepath.Join("backend", "internal", "tool", "picture", clean),
		filepath.Join("..", "tool", "picture", clean),
	}
	for _, candidate := range candidates {
		data, err := os.ReadFile(candidate)
		if err == nil {
			return base64.StdEncoding.EncodeToString(data), nil
		}
	}
	return "", fmt.Errorf("picture %s not found", clean)
}
