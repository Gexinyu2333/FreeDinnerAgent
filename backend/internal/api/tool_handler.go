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
	IdempotencyKey *string         `json:"idempotency_key"`
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

func (h *ToolHandler) Approvals(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}
	status := trimAPIString(queryStringPtr(c, "status"))
	approvals, err := h.tools.ListApprovals(c.Request.Context(), userID, status, parseLimit(c.Query("limit")))
	if err != nil {
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list approval requests")
		return
	}
	OK(c, approvals)
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
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		if errors.Is(err, toolsvc.ErrApprovalRequired) {
			c.JSON(http.StatusAccepted, responseBody{Data: result})
			return
		}
		if errors.Is(err, store.ErrNotFound) {
			Error(c, http.StatusNotFound, "NOT_FOUND", "tool not found")
			return
		}
		Error(c, http.StatusBadRequest, "TOOL_EXECUTION_FAILED", err.Error())
		return
	}
	OK(c, result)
}

func (h *ToolHandler) ConversationCalls(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}
	logs, err := h.tools.ListConversationToolCalls(c.Request.Context(), userID, c.Param("conversation_id"), parseIntQuery(c, "limit", 50))
	if err != nil {
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list tool calls")
		return
	}
	OK(c, gin.H{"tool_calls": logs})
}

func (h *ToolHandler) Call(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}
	log, err := h.tools.GetToolCall(c.Request.Context(), userID, c.Param("tool_call_id"))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			Error(c, http.StatusNotFound, "NOT_FOUND", "tool call not found")
			return
		}
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to load tool call")
		return
	}
	OK(c, log)
}

func (h *ToolHandler) Approve(c *gin.Context) {
	h.resolveApproval(c, "approved")
}

func (h *ToolHandler) Reject(c *gin.Context) {
	h.resolveApproval(c, "rejected")
}

func (h *ToolHandler) resolveApproval(c *gin.Context, status string) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}
	approval, err := h.tools.ResolveApproval(c.Request.Context(), userID, c.Param("approval_id"), status)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			Error(c, http.StatusNotFound, "NOT_FOUND", "approval request not found")
			return
		}
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to resolve approval")
		return
	}
	OK(c, approval)
}
