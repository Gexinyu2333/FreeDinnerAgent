package api

import (
	"errors"
	"net/http"
	"strings"

	marketsvc "freedinner/backend/internal/market"
	"freedinner/backend/internal/store"

	"github.com/gin-gonic/gin"
)

type MarketHandler struct {
	market *marketsvc.Service
}

func NewMarketHandler(market *marketsvc.Service) *MarketHandler {
	return &MarketHandler{market: market}
}

type bindCapabilityRequest struct {
	AgentConfigID   string `json:"agent_config_id"`
	CapabilityType  string `json:"capability_type" binding:"required"`
	CapabilityRefID string `json:"capability_ref_id" binding:"required"`
	LoadMode        string `json:"load_mode"`
	Priority        int    `json:"priority"`
}

type createPromptTemplateRequest struct {
	Name        string                          `json:"name" binding:"required"`
	DisplayName string                          `json:"display_name" binding:"required"`
	Description string                          `json:"description" binding:"required"`
	Category    string                          `json:"category"`
	Tags        []string                        `json:"tags"`
	Visibility  string                          `json:"visibility"`
	Content     string                          `json:"content" binding:"required"`
	ChangeNote  *string                         `json:"change_note"`
	Variables   []promptTemplateVariableRequest `json:"variables"`
}

type promptTemplateVariableRequest struct {
	Name          string   `json:"name" binding:"required"`
	DisplayName   string   `json:"display_name"`
	Description   *string  `json:"description"`
	DefaultValue  *string  `json:"default_value"`
	Required      bool     `json:"required"`
	ValueType     string   `json:"value_type"`
	AllowedValues []string `json:"allowed_values"`
}

type previewPromptTemplateRequest struct {
	VersionID string            `json:"version_id" binding:"required"`
	Variables map[string]string `json:"variables"`
	Override  *string           `json:"override"`
}

type rateMarketplaceItemRequest struct {
	Rating  int     `json:"rating" binding:"required"`
	Comment *string `json:"comment"`
}

func (h *MarketHandler) ListItems(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}
	itemType := trimAPIString(queryStringPtr(c, "item_type"))
	installedOnly := parseBoolDefault(c.Query("installed"), false)
	items, err := h.market.ListItems(c.Request.Context(), userID, itemType, installedOnly, parseLimit(c.Query("limit")))
	if err != nil {
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list marketplace items")
		return
	}
	OK(c, items)
}

func (h *MarketHandler) Install(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}
	install, err := h.market.Install(c.Request.Context(), userID, c.Param("item_id"))
	if err != nil {
		writeMarketError(c, err, "failed to install capability")
		return
	}
	OK(c, install)
}

func (h *MarketHandler) Rate(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}
	var req rateMarketplaceItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	result, err := h.market.RateItem(c.Request.Context(), userID, c.Param("item_id"), req.Rating, req.Comment)
	if err != nil {
		writeMarketError(c, err, "failed to rate capability")
		return
	}
	OK(c, result)
}

func (h *MarketHandler) EnableInstall(c *gin.Context) {
	h.setInstall(c, true)
}

func (h *MarketHandler) DisableInstall(c *gin.Context) {
	h.setInstall(c, false)
}

func (h *MarketHandler) setInstall(c *gin.Context, enabled bool) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}
	install, err := h.market.SetInstallEnabled(c.Request.Context(), userID, c.Param("install_id"), enabled)
	if err != nil {
		writeMarketError(c, err, "failed to update capability install")
		return
	}
	OK(c, install)
}

func (h *MarketHandler) Bind(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}
	var req bindCapabilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	binding, err := h.market.Bind(c.Request.Context(), userID, req.AgentConfigID, req.CapabilityType, req.CapabilityRefID, req.LoadMode, req.Priority)
	if err != nil {
		writeMarketError(c, err, "failed to bind capability")
		return
	}
	OK(c, binding)
}

func (h *MarketHandler) EnableBinding(c *gin.Context) {
	h.setBinding(c, true)
}

func (h *MarketHandler) DisableBinding(c *gin.Context) {
	h.setBinding(c, false)
}

func (h *MarketHandler) setBinding(c *gin.Context, enabled bool) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}
	binding, err := h.market.SetBindingEnabled(c.Request.Context(), userID, c.Param("binding_id"), enabled)
	if err != nil {
		writeMarketError(c, err, "failed to update capability binding")
		return
	}
	OK(c, binding)
}

func (h *MarketHandler) CreatePromptTemplate(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}
	var req createPromptTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", "content is required")
		return
	}
	result, err := h.market.CreatePromptTemplate(c.Request.Context(), marketsvc.CreatePromptTemplateInput{
		UserID:      userID,
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		Category:    req.Category,
		Tags:        req.Tags,
		Visibility:  req.Visibility,
		Content:     req.Content,
		ChangeNote:  req.ChangeNote,
		Variables:   toPromptVariableInputs(req.Variables),
	})
	if err != nil {
		writeMarketError(c, err, "failed to create system prompt template")
		return
	}
	OK(c, result)
}

func (h *MarketHandler) PreviewPromptTemplate(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}
	var req previewPromptTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if req.Variables == nil {
		req.Variables = map[string]string{}
	}
	result, err := h.market.PreviewPrompt(c.Request.Context(), marketsvc.PreviewPromptInput{
		UserID:    userID,
		VersionID: req.VersionID,
		Variables: req.Variables,
		Override:  req.Override,
	})
	if err != nil {
		writeMarketError(c, err, "failed to preview system prompt template")
		return
	}
	OK(c, result)
}

func writeMarketError(c *gin.Context, err error, fallback string) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		Error(c, http.StatusNotFound, "NOT_FOUND", "market capability not found")
	case errors.Is(err, marketsvc.ErrInvalidInput):
		Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
	default:
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", fallback)
	}
}

func toPromptVariableInputs(values []promptTemplateVariableRequest) []marketsvc.PromptVariableInput {
	result := make([]marketsvc.PromptVariableInput, 0, len(values))
	for _, value := range values {
		result = append(result, marketsvc.PromptVariableInput{
			Name:          value.Name,
			DisplayName:   value.DisplayName,
			Description:   value.Description,
			DefaultValue:  value.DefaultValue,
			Required:      value.Required,
			ValueType:     value.ValueType,
			AllowedValues: value.AllowedValues,
		})
	}
	return result
}
