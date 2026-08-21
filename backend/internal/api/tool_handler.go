package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"freedinner/backend/internal/store"
	toolsvc "freedinner/backend/internal/tool"

	"github.com/gin-gonic/gin"
)

type ToolHandler struct {
	tools *toolsvc.Service
}

func NewToolHandler(tools *toolsvc.Service) *ToolHandler {
	return &ToolHandler{tools: tools}
}

type executeToolRequest struct {
	ConversationID string          `json:"conversation_id" binding:"required"`
	Arguments      json.RawMessage `json:"arguments"`
}

func (h *ToolHandler) List(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}

	tools, err := h.tools.ListTools(c.Request.Context(), userID)
	if err != nil {
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list tools")
		return
	}
	OK(c, tools)
}

func (h *ToolHandler) Execute(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}

	var req executeToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if strings.TrimSpace(req.ConversationID) == "" {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", "conversation_id is required")
		return
	}
	if len(req.Arguments) == 0 {
		req.Arguments = json.RawMessage(`{}`)
	}

	result, err := h.tools.Execute(c.Request.Context(), toolsvc.ExecuteInput{
		UserID:         userID,
		ConversationID: req.ConversationID,
		ToolName:       c.Param("tool_name"),
		Arguments:      req.Arguments,
	})
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			Error(c, http.StatusNotFound, "NOT_FOUND", "tool not found")
			return
		}
		Error(c, http.StatusBadRequest, "TOOL_EXECUTION_FAILED", err.Error())
		return
	}
	OK(c, result)
}
