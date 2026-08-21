package api

import (
	"net/http"

	"freedinner/backend/internal/contextmgr"
	"freedinner/backend/internal/store"

	"github.com/gin-gonic/gin"
)

type ContextHandler struct {
	conversations *store.ConversationStore
	compressor    *contextmgr.Compressor
}

func NewContextHandler(conversations *store.ConversationStore, compressor *contextmgr.Compressor) *ContextHandler {
	return &ContextHandler{conversations: conversations, compressor: compressor}
}

type manualCompressRequest struct {
	KeepRecentTurns   int    `json:"keep_recent_turns"`
	TargetSummaryType string `json:"target_summary_type"`
}

func (h *ContextHandler) ManualCompress(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}
	var req manualCompressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if req.KeepRecentTurns <= 0 {
		req.KeepRecentTurns = contextmgr.DefaultRecentTurnLimit
	}
	if req.KeepRecentTurns > 50 {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", "keep_recent_turns must be <= 50")
		return
	}
	if req.TargetSummaryType == "" {
		req.TargetSummaryType = "turn_window"
	}
	if req.TargetSummaryType != "turn_window" && req.TargetSummaryType != "session" && req.TargetSummaryType != "handoff" {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", "target_summary_type is invalid")
		return
	}

	conversationID := c.Param("conversation_id")
	messages, err := h.conversations.ListMessages(c.Request.Context(), userID, conversationID)
	if err != nil {
		if err == store.ErrNotFound {
			Error(c, http.StatusNotFound, "NOT_FOUND", "conversation not found")
			return
		}
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list conversation messages")
		return
	}
	result, err := h.compressor.ManualCompress(c.Request.Context(), contextmgr.ManualCompressInput{
		UserID:            userID,
		ConversationID:    conversationID,
		Messages:          messages,
		KeepRecentTurns:   req.KeepRecentTurns,
		TargetSummaryType: req.TargetSummaryType,
	})
	if err != nil {
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to compress conversation")
		return
	}
	OK(c, result)
}
