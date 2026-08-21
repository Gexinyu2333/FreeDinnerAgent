package api

import (
	"errors"
	"net/http"

	"freedinner/backend/internal/store"

	"github.com/gin-gonic/gin"
)

type HarnessHandler struct {
	harness *store.HarnessStore
}

func NewHarnessHandler(harness *store.HarnessStore) *HarnessHandler {
	return &HarnessHandler{harness: harness}
}

func (h *HarnessHandler) Events(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}

	var turnID *string
	if value := c.Query("turn_id"); value != "" {
		turnID = &value
	}

	events, err := h.harness.ListEvents(c.Request.Context(), userID, c.Param("conversation_id"), turnID)
	if err != nil {
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list agent events")
		return
	}
	OK(c, events)
}

func (h *HarnessHandler) Turn(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}

	turn, err := h.harness.GetTurn(c.Request.Context(), userID, c.Param("conversation_id"), c.Param("turn_id"))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			Error(c, http.StatusNotFound, "NOT_FOUND", "agent turn not found")
			return
		}
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get agent turn")
		return
	}
	OK(c, turn)
}

func (h *HarnessHandler) LoopSteps(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}

	steps, err := h.harness.ListLoopSteps(c.Request.Context(), userID, c.Param("conversation_id"), c.Param("turn_id"))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			Error(c, http.StatusNotFound, "NOT_FOUND", "agent turn not found")
			return
		}
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list loop steps")
		return
	}
	OK(c, steps)
}
