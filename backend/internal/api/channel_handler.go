package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	channelsvc "freedinner/backend/internal/channel"
	"freedinner/backend/internal/store"

	"github.com/gin-gonic/gin"
)

type ChannelHandler struct {
	channels *channelsvc.Service
}

func NewChannelHandler(channels *channelsvc.Service) *ChannelHandler {
	return &ChannelHandler{channels: channels}
}

type createChannelConnectionRequest struct {
	ProviderID          string            `json:"provider_id" binding:"required"`
	DisplayName         string            `json:"display_name" binding:"required"`
	ExternalAccountID   *string           `json:"external_account_id"`
	ExternalAccountName *string           `json:"external_account_name"`
	Endpoints           []endpointRequest `json:"endpoints"`
	Config              json.RawMessage   `json:"config"`
}

type endpointRequest struct {
	EndpointType string          `json:"endpoint_type"`
	DisplayName  string          `json:"display_name"`
	Direction    string          `json:"direction"`
	Transport    string          `json:"transport"`
	URL          string          `json:"url"`
	Config       json.RawMessage `json:"config"`
	Metadata     json.RawMessage `json:"metadata"`
}

type upsertChannelPolicyRequest struct {
	ScopeType                  string          `json:"scope_type" binding:"required"`
	ExternalScopeID            *string         `json:"external_scope_id"`
	Mode                       string          `json:"mode" binding:"required"`
	TriggerKeywords            []string        `json:"trigger_keywords"`
	AllowMemoryWrite           *bool           `json:"allow_memory_write"`
	AllowToolUse               *bool           `json:"allow_tool_use"`
	RequireApprovalForOutbound *bool           `json:"require_approval_for_outbound"`
	RateLimitPerMinute         *int            `json:"rate_limit_per_minute"`
	RateLimitPolicy            json.RawMessage `json:"rate_limit_policy"`
}

func (h *ChannelHandler) Providers(c *gin.Context) {
	providers, err := h.channels.ListProviders(c.Request.Context())
	if err != nil {
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list channel providers")
		return
	}
	OK(c, providers)
}

func (h *ChannelHandler) CreateConnection(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}

	var req createChannelConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if strings.TrimSpace(req.DisplayName) == "" {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", "display_name is required")
		return
	}

	connection, err := h.channels.CreateConnection(c.Request.Context(), channelsvc.CreateConnectionInput{
		UserID:              userID,
		ProviderID:          req.ProviderID,
		DisplayName:         req.DisplayName,
		ExternalAccountID:   req.ExternalAccountID,
		ExternalAccountName: req.ExternalAccountName,
		Endpoints:           toEndpointInputs(req.Endpoints),
		Config:              req.Config,
	})
	if err != nil {
		if isUniqueViolation(err) {
			Error(c, http.StatusConflict, "CHANNEL_CONNECTION_EXISTS", "channel connection already exists")
			return
		}
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create channel connection")
		return
	}
	data, err := h.publicConnection(c.Request.Context(), userID, connection)
	if err != nil {
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to load channel connection endpoints")
		return
	}
	OK(c, data)
}

func (h *ChannelHandler) Connections(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}

	connections, err := h.channels.ListConnections(c.Request.Context(), userID)
	if err != nil {
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list channel connections")
		return
	}
	data := make([]store.PublicChannelConnection, 0, len(connections))
	for _, connection := range connections {
		item, err := h.publicConnection(c.Request.Context(), userID, connection)
		if err != nil {
			Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to load channel connection endpoints")
			return
		}
		data = append(data, item)
	}
	OK(c, data)
}

func (h *ChannelHandler) publicConnection(ctx context.Context, userID string, connection store.ChannelConnection) (store.PublicChannelConnection, error) {
	data := store.ToPublicChannelConnection(connection)
	endpoints, err := h.channels.ListEndpoints(ctx, userID, connection.ID)
	if err != nil {
		return store.PublicChannelConnection{}, err
	}
	data.Endpoints = make([]store.PublicChannelConnectionEndpoint, 0, len(endpoints))
	for _, endpoint := range endpoints {
		data.Endpoints = append(data.Endpoints, store.ToPublicChannelConnectionEndpoint(endpoint))
	}
	return data, nil
}

func toEndpointInputs(items []endpointRequest) []channelsvc.EndpointInput {
	result := make([]channelsvc.EndpointInput, 0, len(items))
	for _, item := range items {
		result = append(result, channelsvc.EndpointInput{
			EndpointType: item.EndpointType,
			DisplayName:  item.DisplayName,
			Direction:    item.Direction,
			Transport:    item.Transport,
			URL:          item.URL,
			SecretConfig: item.Config,
			Metadata:     item.Metadata,
		})
	}
	return result
}

func (h *ChannelHandler) UpsertPolicy(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}

	var req upsertChannelPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	policy, err := h.channels.UpsertPolicy(c.Request.Context(), channelsvc.UpsertPolicyInput{
		UserID:                     userID,
		ConnectionID:               c.Param("connection_id"),
		ScopeType:                  req.ScopeType,
		ExternalScopeID:            req.ExternalScopeID,
		Mode:                       req.Mode,
		TriggerKeywords:            req.TriggerKeywords,
		AllowMemoryWrite:           boolDefault(req.AllowMemoryWrite, true),
		AllowToolUse:               boolDefault(req.AllowToolUse, true),
		RequireApprovalForOutbound: boolDefault(req.RequireApprovalForOutbound, true),
		RateLimitPerMinute:         intDefault(req.RateLimitPerMinute, 6),
		RateLimitPolicy:            req.RateLimitPolicy,
	})
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			Error(c, http.StatusNotFound, "NOT_FOUND", "channel connection not found")
			return
		}
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to upsert channel policy")
		return
	}
	OK(c, policy)
}

func (h *ChannelHandler) Policies(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}
	policies, err := h.channels.ListPolicies(c.Request.Context(), userID, c.Param("connection_id"))
	writeChannelList(c, policies, err)
}

func (h *ChannelHandler) Webhook(c *gin.Context) {
	secret := webhookSecret(c)
	raw, err := c.GetRawData()
	if err != nil {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", "failed to read request body")
		return
	}

	result, err := h.channels.HandleWebhook(c.Request.Context(), c.Param("connection_id"), secret, raw)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			Error(c, http.StatusNotFound, "NOT_FOUND", "channel connection not found")
		case errors.Is(err, channelsvc.ErrInvalidWebhookSecret):
			Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid webhook secret")
		default:
			Error(c, http.StatusBadRequest, "WEBHOOK_FAILED", err.Error())
		}
		return
	}
	OK(c, result)
}

func webhookSecret(c *gin.Context) string {
	for _, header := range []string{"X-FreeDinner-Webhook-Secret", "X-Access-Token", "X-Token"} {
		if secret := strings.TrimSpace(c.GetHeader(header)); secret != "" {
			return secret
		}
	}
	auth := strings.TrimSpace(c.GetHeader("Authorization"))
	for _, prefix := range []string{"Bearer ", "Token "} {
		if strings.HasPrefix(auth, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(auth, prefix))
		}
	}
	if auth != "" {
		return auth
	}
	for _, key := range []string{"access_token", "token", "secret"} {
		if secret := strings.TrimSpace(c.Query(key)); secret != "" {
			return secret
		}
	}
	return ""
}

func (h *ChannelHandler) ExternalConversations(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}
	items, err := h.channels.ListExternalConversations(c.Request.Context(), userID, c.Param("connection_id"), parseLimit(c.Query("limit")))
	writeChannelList(c, items, err)
}

func (h *ChannelHandler) InboxEvents(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}
	items, err := h.channels.ListInboxEvents(c.Request.Context(), userID, c.Param("connection_id"), parseLimit(c.Query("limit")))
	writeChannelList(c, items, err)
}

func (h *ChannelHandler) OutboxMessages(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}
	status := trimAPIString(queryStringPtr(c, "status"))
	items, err := h.channels.ListOutboxMessages(c.Request.Context(), userID, c.Param("connection_id"), status, parseLimit(c.Query("limit")))
	writeChannelList(c, items, err)
}

func (h *ChannelHandler) ApproveOutbox(c *gin.Context) {
	h.resolveOutbox(c, "approve")
}

func (h *ChannelHandler) CancelOutbox(c *gin.Context) {
	h.resolveOutbox(c, "cancel")
}

func (h *ChannelHandler) SendOutbox(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}
	message, err := h.channels.SendOutboxMessage(c.Request.Context(), userID, c.Param("outbox_id"))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			Error(c, http.StatusNotFound, "NOT_FOUND", "approved outbox message not found")
			return
		}
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to send outbox message")
		return
	}
	OK(c, message)
}

func (h *ChannelHandler) resolveOutbox(c *gin.Context, action string) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}
	var (
		message store.ChannelOutboxMessage
		err     error
	)
	switch action {
	case "approve":
		message, err = h.channels.ApproveOutboxMessage(c.Request.Context(), userID, c.Param("outbox_id"))
	case "cancel":
		message, err = h.channels.CancelOutboxMessage(c.Request.Context(), userID, c.Param("outbox_id"))
	}
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			Error(c, http.StatusNotFound, "NOT_FOUND", "pending outbox message not found")
			return
		}
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update outbox message")
		return
	}
	OK(c, message)
}

func writeChannelList(c *gin.Context, data any, err error) {
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			Error(c, http.StatusNotFound, "NOT_FOUND", "channel connection not found")
			return
		}
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list channel data")
		return
	}
	OK(c, data)
}

func boolDefault(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func intDefault(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}
