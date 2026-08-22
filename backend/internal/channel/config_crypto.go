package channel

import (
	"encoding/json"
	"strings"
)

type connectionConfig struct {
	AccessToken   string `json:"access_token"`
	WebhookSecret string `json:"webhook_secret"`
	BotQQ         string `json:"bot_qq"`
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
