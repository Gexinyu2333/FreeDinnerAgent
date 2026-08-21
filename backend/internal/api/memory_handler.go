package api

import (
	"net/http"
	"strconv"
	"strings"

	"freedinner/backend/internal/store"

	"github.com/gin-gonic/gin"
)

type MemoryHandler struct {
	memories *store.MemoryStore
}

func NewMemoryHandler(memories *store.MemoryStore) *MemoryHandler {
	return &MemoryHandler{memories: memories}
}

type createProfileMemoryRequest struct {
	MemoryType string  `json:"memory_type" binding:"required"`
	Scope      string  `json:"scope"`
	Title      string  `json:"title" binding:"required"`
	Content    string  `json:"content" binding:"required"`
	Evidence   *string `json:"evidence"`
	Confidence float64 `json:"confidence"`
	Importance int     `json:"importance"`
}

func (h *MemoryHandler) Types(c *gin.Context) {
	types, err := h.memories.ListTypes(c.Request.Context())
	if err != nil {
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list memory types")
		return
	}
	OK(c, types)
}

func (h *MemoryHandler) CreateProfileMemory(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}

	var req createProfileMemoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if err := validateCreateProfileMemory(req); err != nil {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	memory, err := h.memories.CreateProfileMemory(c.Request.Context(), store.ProfileMemoryCreate{
		UserID:     userID,
		MemoryType: strings.TrimSpace(req.MemoryType),
		Scope:      normalizeMemoryScope(req.Scope),
		Title:      strings.TrimSpace(req.Title),
		Content:    strings.TrimSpace(req.Content),
		Evidence:   normalizeOptionalString(req.Evidence),
		Confidence: req.Confidence,
		Importance: req.Importance,
	})
	if err != nil {
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create profile memory")
		return
	}
	OK(c, memory)
}

func (h *MemoryHandler) ListProfileMemories(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}

	var memoryType *string
	if value := strings.TrimSpace(c.Query("memory_type")); value != "" {
		memoryType = &value
	}

	memories, err := h.memories.ListProfileMemories(c.Request.Context(), userID, memoryType)
	if err != nil {
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list profile memories")
		return
	}
	OK(c, memories)
}

func (h *MemoryHandler) SearchProfileMemories(c *gin.Context) {
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
	if limit <= 0 || limit > 20 {
		limit = 8
	}

	memories, err := h.memories.SearchProfileMemories(c.Request.Context(), userID, query, limit)
	if err != nil {
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to search profile memories")
		return
	}
	OK(c, gin.H{
		"mode":     "keyword",
		"memories": memories,
	})
}

func validateCreateProfileMemory(req createProfileMemoryRequest) error {
	if strings.TrimSpace(req.MemoryType) == "" {
		return errMessage("memory_type is required")
	}
	if strings.TrimSpace(req.Title) == "" {
		return errMessage("title is required")
	}
	if strings.TrimSpace(req.Content) == "" {
		return errMessage("content is required")
	}
	if req.Confidence != 0 && (req.Confidence < 0 || req.Confidence > 1) {
		return errMessage("confidence must be between 0 and 1")
	}
	if req.Importance != 0 && (req.Importance < 1 || req.Importance > 5) {
		return errMessage("importance must be between 1 and 5")
	}
	return nil
}

func normalizeMemoryScope(value string) string {
	switch value {
	case "project", "conversation":
		return value
	default:
		return "global"
	}
}

type errMessage string

func (e errMessage) Error() string {
	return string(e)
}
