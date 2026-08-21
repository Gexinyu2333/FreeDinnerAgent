package api

import (
	"errors"
	"net/http"
	"strings"

	"freedinner/backend/internal/llm"
	"freedinner/backend/internal/store"

	"github.com/gin-gonic/gin"
)

type ConversationHandler struct {
	conversations *store.ConversationStore
	llm           *llm.Service
}

func NewConversationHandler(conversations *store.ConversationStore, llmService *llm.Service) *ConversationHandler {
	return &ConversationHandler{
		conversations: conversations,
		llm:           llmService,
	}
}

type createConversationRequest struct {
	Title string `json:"title"`
}

type sendMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

type sendMessageResponse struct {
	TurnID           *string       `json:"turn_id"`
	UserMessage      store.Message `json:"user_message"`
	AssistantMessage store.Message `json:"assistant_message"`
	UsedMemories     []any         `json:"used_memories"`
	ToolCalls        []any         `json:"tool_calls"`
}

func (h *ConversationHandler) Create(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}

	var req createConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "新的对话"
	}
	if len(title) > 200 {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", "title is too long")
		return
	}

	conversation, err := h.conversations.Create(c.Request.Context(), userID, title)
	if err != nil {
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create conversation")
		return
	}

	OK(c, conversation)
}

func (h *ConversationHandler) List(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}

	conversations, err := h.conversations.List(c.Request.Context(), userID)
	if err != nil {
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list conversations")
		return
	}

	OK(c, conversations)
}

func (h *ConversationHandler) Messages(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}

	messages, err := h.conversations.ListMessages(c.Request.Context(), userID, c.Param("conversation_id"))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			Error(c, http.StatusNotFound, "NOT_FOUND", "conversation not found")
			return
		}
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list messages")
		return
	}

	OK(c, messages)
}

func (h *ConversationHandler) SendMessage(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}

	var req sendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	content := strings.TrimSpace(req.Content)
	if content == "" {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", "content is required")
		return
	}

	result, err := h.llm.SendMessage(c.Request.Context(), userID, c.Param("conversation_id"), content)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			Error(c, http.StatusNotFound, "NOT_FOUND", "conversation not found")
			return
		case errors.Is(err, llm.ErrModelProviderRequired):
			Error(c, http.StatusBadRequest, "MODEL_PROVIDER_REQUIRED", "请先在设置中配置 OpenAI 或 Anthropic API Key")
			return
		case errors.Is(err, llm.ErrUnsupportedProvider):
			Error(c, http.StatusBadRequest, "UNSUPPORTED_PROVIDER", "当前阶段仅支持 OpenAI 模型供应商")
			return
		case errors.Is(err, llm.ErrLLMCallFailed):
			Error(c, http.StatusBadGateway, "LLM_CALL_FAILED", "模型调用失败，请检查 API Key、模型名称或网络状态")
			return
		}
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to send message")
		return
	}

	OK(c, sendMessageResponse{
		TurnID:           &result.TurnID,
		UserMessage:      result.UserMessage,
		AssistantMessage: result.AssistantMessage,
		UsedMemories:     []any{},
		ToolCalls:        []any{},
	})
}
