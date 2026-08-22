package tool

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"freedinner/backend/internal/store"
)

func (s *Service) executeMCPTool(ctx context.Context, toolDefinition store.ToolDefinition, input ExecuteInput) (json.RawMessage, string, *string, *string) {
	var metadata struct {
		Endpoint      *string `json:"endpoint"`
		TransportType string  `json:"transport_type"`
		HandlerRef    string  `json:"handler_ref"`
	}
	_ = json.Unmarshal(toolDefinition.Metadata, &metadata)
	if metadata.Endpoint == nil || strings.TrimSpace(*metadata.Endpoint) == "" {
		message := "mcp tool requires http bridge endpoint"
		errorType := "unsupported_mcp_transport"
		raw, _ := json.Marshal(map[string]any{"error": message})
		return raw, "failed", &errorType, &message
	}
	endpoint := strings.TrimRight(strings.TrimSpace(*metadata.Endpoint), "/")
	handlerRef := strings.TrimSpace(metadata.HandlerRef)
	if handlerRef == "" {
		handlerRef = toolDefinition.HandlerRef
	}
	payload, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      stableRequestID(input.IdempotencyKey, input.ToolName),
		"method":  "tools/call",
		"params": map[string]any{
			"name":      handlerRef,
			"arguments": json.RawMessage(input.Arguments),
		},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		message := err.Error()
		errorType := "mcp_request_error"
		raw, _ := json.Marshal(map[string]any{"error": message})
		return raw, "failed", &errorType, &message
	}
	req.Header.Set("Content-Type", "application/json")
	client := s.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		message := err.Error()
		errorType := "mcp_transport_error"
		raw, _ := json.Marshal(map[string]any{"error": message})
		return raw, "failed", &errorType, &message
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := "mcp bridge returned status " + resp.Status
		errorType := "mcp_status_error"
		raw, _ := json.Marshal(map[string]any{"error": message, "body": string(body)})
		return raw, "failed", &errorType, &message
	}
	if !json.Valid(body) {
		raw, _ := json.Marshal(map[string]any{"content": string(body)})
		return raw, "success", nil, nil
	}
	return json.RawMessage(body), "success", nil, nil
}

func stableRequestID(idempotencyKey *string, toolName string) string {
	source := toolName + ":" + time.Now().UTC().Format(time.RFC3339Nano)
	if idempotencyKey != nil && strings.TrimSpace(*idempotencyKey) != "" {
		source = *idempotencyKey
	}
	sum := sha256.Sum256([]byte(source))
	return hex.EncodeToString(sum[:8])
}
