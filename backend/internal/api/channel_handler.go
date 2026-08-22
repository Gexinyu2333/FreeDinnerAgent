package api

import (
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
	ProviderID          string          `json:"provider_id" binding:"required"`
	DisplayName         string          `json:"display_name" binding:"required"`
	ExternalAccountID   *string         `json:"external_account_id"`
	ExternalAccountName *string         `json:"external_account_name"`
	Config              json.RawMessage `json:"config"`
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
	OK(c, store.ToPublicChannelConnection(connection))
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
		data = append(data, store.ToPublicChannelConnection(connection))
	}
	OK(c, data)
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
	secret := c.GetHeader("X-FreeDinner-Webhook-Secret")
	if secret == "" {
		secret = strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
	}
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
