package api

import (
	"net/http"
	"strconv"
	"strings"

	"freedinner/backend/internal/knowledge"

	"github.com/gin-gonic/gin"
)

type KnowledgeHandler struct {
	knowledge *knowledge.Service
}

func NewKnowledgeHandler(knowledgeService *knowledge.Service) *KnowledgeHandler {
	return &KnowledgeHandler{knowledge: knowledgeService}
}

type ingestKnowledgeRequest struct {
	Title      string  `json:"title" binding:"required"`
	Content    string  `json:"content" binding:"required"`
	SourceType string  `json:"source_type"`
	SourceURI  *string `json:"source_uri"`
	Visibility string  `json:"visibility"`
}

func (h *KnowledgeHandler) Ingest(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}

	var req ingestKnowledgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	title := strings.TrimSpace(req.Title)
	content := strings.TrimSpace(req.Content)
	if title == "" {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", "title is required")
		return
	}
	if content == "" {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", "content is required")
		return
	}

	result, err := h.knowledge.Ingest(c.Request.Context(), knowledge.IngestInput{
		UserID:     userID,
		Title:      title,
		Content:    content,
		SourceType: req.SourceType,
		SourceURI:  normalizeOptionalString(req.SourceURI),
		Visibility: req.Visibility,
	})
	if err != nil {
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to ingest knowledge document")
		return
	}

	OK(c, result)
}

func (h *KnowledgeHandler) ListDocuments(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}

	documents, err := h.knowledge.ListDocuments(c.Request.Context(), userID)
	if err != nil {
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list knowledge documents")
		return
	}
	OK(c, documents)
}

func (h *KnowledgeHandler) Search(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}

	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", "q is required")
		return
	}

	limit := 8
	if value := c.Query("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			Error(c, http.StatusBadRequest, "BAD_REQUEST", "limit must be a number")
			return
		}
		limit = parsed
	}

	result, err := h.knowledge.Search(c.Request.Context(), userID, query, limit)
	if err != nil {
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to search knowledge")
		return
	}
	OK(c, result)
}
