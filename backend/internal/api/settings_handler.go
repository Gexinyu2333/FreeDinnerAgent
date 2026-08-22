package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"freedinner/backend/internal/secret"
	"freedinner/backend/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
)

type SettingsHandler struct {
	agentConfigs   *store.AgentConfigStore
	modelProviders *store.ModelProviderStore
	crypto         secret.Crypto
}

func NewSettingsHandler(agentConfigs *store.AgentConfigStore, modelProviders *store.ModelProviderStore, crypto secret.Crypto) *SettingsHandler {
	return &SettingsHandler{
		agentConfigs:   agentConfigs,
		modelProviders: modelProviders,
		crypto:         crypto,
	}
}

type updateAgentConfigRequest struct {
	Name                  *string                          `json:"name"`
	SystemPrompt          *string                          `json:"system_prompt"`
	DefaultProviderID     **string                         `json:"default_provider_id"`
	Temperature           *float64                         `json:"temperature"`
	ThinkingEnabled       *bool                            `json:"thinking_enabled"`
	ThinkingEffort        *string                          `json:"thinking_effort"`
	ThinkingBudgetTokens  *int                             `json:"thinking_budget_tokens"`
	MaxContextTokens      *int                             `json:"max_context_tokens"`
	MaxLoopSteps          *int                             `json:"max_loop_steps"`
	LLMRetryLimit         *int                             `json:"llm_retry_limit"`
	FallbackPolicy        *json.RawMessage                 `json:"fallback_policy"`
	MemoryEnabled         *bool                            `json:"memory_enabled"`
	ToolUseEnabled        *bool                            `json:"tool_use_enabled"`
	ToolApprovalPolicy    *string                          `json:"tool_approval_policy"`
	DreamingEnabled       *bool                            `json:"dreaming_enabled"`
	SemanticMemoryEnabled *bool                            `json:"semantic_memory_enabled"`
	EmbeddingEnabled      *bool                            `json:"embedding_enabled"`
	EmbeddingCostPolicy   *json.RawMessage                 `json:"embedding_cost_policy"`
	LLMFeatureSettings    *[]store.LLMFeatureSettingUpdate `json:"llm_feature_settings"`
}

func (h *SettingsHandler) GetAgentConfig(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}

	cfg, err := h.agentConfigs.GetDefault(c.Request.Context(), userID)
	if err != nil {
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to load agent config")
		return
	}

	OK(c, cfg)
}

func (h *SettingsHandler) UpdateAgentConfig(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}

	var req updateAgentConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if err := validateAgentConfigRequest(req); err != nil {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	cfg, err := h.agentConfigs.UpdateDefault(c.Request.Context(), userID, store.AgentConfigUpdate{
		Name:                  req.Name,
		SystemPrompt:          req.SystemPrompt,
		DefaultProviderID:     req.DefaultProviderID,
		Temperature:           req.Temperature,
		ThinkingEnabled:       req.ThinkingEnabled,
		ThinkingEffort:        req.ThinkingEffort,
		ThinkingBudgetTokens:  req.ThinkingBudgetTokens,
		MaxContextTokens:      req.MaxContextTokens,
		MaxLoopSteps:          req.MaxLoopSteps,
		LLMRetryLimit:         req.LLMRetryLimit,
		FallbackPolicy:        req.FallbackPolicy,
		MemoryEnabled:         req.MemoryEnabled,
		ToolUseEnabled:        req.ToolUseEnabled,
		ToolApprovalPolicy:    req.ToolApprovalPolicy,
		DreamingEnabled:       req.DreamingEnabled,
		SemanticMemoryEnabled: req.SemanticMemoryEnabled,
		EmbeddingEnabled:      req.EmbeddingEnabled,
		EmbeddingCostPolicy:   req.EmbeddingCostPolicy,
		LLMFeatureSettings:    req.LLMFeatureSettings,
	})
	if err != nil {
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update agent config")
		return
	}

	OK(c, cfg)
}

type createModelProviderRequest struct {
	Provider              string  `json:"provider" binding:"required"`
	DisplayName           string  `json:"display_name" binding:"required"`
	ChatBaseURL           *string `json:"chat_base_url"`
	ChatAPIKey            string  `json:"chat_api_key" binding:"required"`
	EmbeddingBaseURL      *string `json:"embedding_base_url"`
	EmbeddingAPIKey       *string `json:"embedding_api_key"`
	DefaultChatModel      string  `json:"default_chat_model" binding:"required"`
	DefaultEmbeddingModel *string `json:"default_embedding_model"`
	IsDefault             bool    `json:"is_default"`
}

type updateModelProviderRequest struct {
	DisplayName           *string  `json:"display_name"`
	ChatBaseURL           **string `json:"chat_base_url"`
	ChatAPIKey            *string  `json:"chat_api_key"`
	EmbeddingBaseURL      **string `json:"embedding_base_url"`
	EmbeddingAPIKey       **string `json:"embedding_api_key"`
	DefaultChatModel      *string  `json:"default_chat_model"`
	DefaultEmbeddingModel **string `json:"default_embedding_model"`
	IsDefault             *bool    `json:"is_default"`
	Status                *string  `json:"status"`
}

func (h *SettingsHandler) ListModelProviders(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}

	providers, err := h.modelProviders.List(c.Request.Context(), userID)
	if err != nil {
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list model providers")
		return
	}

	data := make([]store.PublicModelProvider, 0, len(providers))
	for _, provider := range providers {
		data = append(data, store.ToPublicModelProvider(provider))
	}
	OK(c, data)
}

func (h *SettingsHandler) CreateModelProvider(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}

	var req createModelProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if err := validateModelProviderCreate(req); err != nil {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	encryptedChatAPIKey, err := h.crypto.Encrypt(req.ChatAPIKey)
	if err != nil {
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to encrypt chat api key")
		return
	}
	var encryptedEmbeddingAPIKey *string
	if req.EmbeddingAPIKey != nil && strings.TrimSpace(*req.EmbeddingAPIKey) != "" {
		value, err := h.crypto.Encrypt(*req.EmbeddingAPIKey)
		if err != nil {
			Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to encrypt embedding api key")
			return
		}
		encryptedEmbeddingAPIKey = &value
	}

	provider, err := h.modelProviders.Create(c.Request.Context(), userID, store.ModelProviderCreate{
		Provider:                 req.Provider,
		DisplayName:              strings.TrimSpace(req.DisplayName),
		ChatBaseURL:              normalizeOptionalString(req.ChatBaseURL),
		EncryptedChatAPIKey:      encryptedChatAPIKey,
		EmbeddingBaseURL:         normalizeOptionalString(req.EmbeddingBaseURL),
		EncryptedEmbeddingAPIKey: encryptedEmbeddingAPIKey,
		DefaultChatModel:         strings.TrimSpace(req.DefaultChatModel),
		DefaultEmbeddingModel:    normalizeOptionalString(req.DefaultEmbeddingModel),
		IsDefault:                req.IsDefault,
	})
	if err != nil {
		if isUniqueViolation(err) {
			Error(c, http.StatusConflict, "MODEL_PROVIDER_EXISTS", "model provider display name already exists")
			return
		}
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create model provider")
		return
	}

	OK(c, store.ToPublicModelProvider(provider))
}

func (h *SettingsHandler) UpdateModelProvider(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}
	providerID := c.Param("provider_id")

	var req updateModelProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if err := validateModelProviderUpdate(req); err != nil {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	var encryptedChatAPIKey *string
	if req.ChatAPIKey != nil {
		value, err := h.crypto.Encrypt(*req.ChatAPIKey)
		if err != nil {
			Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to encrypt chat api key")
			return
		}
		encryptedChatAPIKey = &value
	}
	var encryptedEmbeddingAPIKey **string
	if req.EmbeddingAPIKey != nil {
		if *req.EmbeddingAPIKey == nil || strings.TrimSpace(**req.EmbeddingAPIKey) == "" {
			var empty *string
			encryptedEmbeddingAPIKey = &empty
		} else {
			value, err := h.crypto.Encrypt(**req.EmbeddingAPIKey)
			if err != nil {
				Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to encrypt embedding api key")
				return
			}
			encrypted := &value
			encryptedEmbeddingAPIKey = &encrypted
		}
	}

	provider, err := h.modelProviders.Update(c.Request.Context(), userID, providerID, store.ModelProviderUpdate{
		DisplayName:              trimStringPtr(req.DisplayName),
		ChatBaseURL:              normalizeOptionalStringPtr(req.ChatBaseURL),
		EncryptedChatAPIKey:      encryptedChatAPIKey,
		EmbeddingBaseURL:         normalizeOptionalStringPtr(req.EmbeddingBaseURL),
		EncryptedEmbeddingAPIKey: encryptedEmbeddingAPIKey,
		DefaultChatModel:         trimStringPtr(req.DefaultChatModel),
		DefaultEmbeddingModel:    normalizeOptionalStringPtr(req.DefaultEmbeddingModel),
		IsDefault:                req.IsDefault,
		Status:                   req.Status,
	})
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			Error(c, http.StatusNotFound, "NOT_FOUND", "model provider not found")
			return
		}
		if isUniqueViolation(err) {
			Error(c, http.StatusConflict, "MODEL_PROVIDER_EXISTS", "model provider display name already exists")
			return
		}
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update model provider")
		return
	}

	OK(c, store.ToPublicModelProvider(provider))
}

func (h *SettingsHandler) DeleteModelProvider(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}

	if err := h.modelProviders.Delete(c.Request.Context(), userID, c.Param("provider_id")); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			Error(c, http.StatusNotFound, "NOT_FOUND", "model provider not found")
			return
		}
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete model provider")
		return
	}

	OK(c, gin.H{"deleted": true})
}

func validateAgentConfigRequest(req updateAgentConfigRequest) error {
	if req.Temperature != nil && (*req.Temperature < 0 || *req.Temperature > 2) {
		return errors.New("temperature must be between 0 and 2")
	}
	if req.ThinkingEffort != nil {
		switch *req.ThinkingEffort {
		case "low", "medium", "high":
		default:
			return errors.New("thinking_effort must be low, medium or high")
		}
	}
	if req.ThinkingBudgetTokens != nil && *req.ThinkingBudgetTokens < 0 {
		return errors.New("thinking_budget_tokens must be greater than or equal to 0")
	}
	if req.MaxContextTokens != nil && *req.MaxContextTokens <= 0 {
		return errors.New("max_context_tokens must be greater than 0")
	}
	if req.MaxLoopSteps != nil && *req.MaxLoopSteps <= 0 {
		return errors.New("max_loop_steps must be greater than 0")
	}
	if req.LLMRetryLimit != nil && *req.LLMRetryLimit < 0 {
		return errors.New("llm_retry_limit must be greater than or equal to 0")
	}
	if req.ToolApprovalPolicy != nil {
		switch *req.ToolApprovalPolicy {
		case "never", "sensitive_only", "always":
		default:
			return errors.New("tool_approval_policy must be never, sensitive_only or always")
		}
	}
	if req.LLMFeatureSettings != nil {
		for _, setting := range *req.LLMFeatureSettings {
			if setting.FeatureKey == "" {
				return errors.New("llm feature_key is required")
			}
			if setting.Temperature != nil && *setting.Temperature != nil && (**setting.Temperature < 0 || **setting.Temperature > 2) {
				return errors.New("llm feature temperature must be between 0 and 2")
			}
		}
	}
	return nil
}

func validateModelProviderCreate(req createModelProviderRequest) error {
	if req.Provider != "openai" && req.Provider != "anthropic" {
		return errors.New("provider must be openai or anthropic")
	}
	if strings.TrimSpace(req.DisplayName) == "" {
		return errors.New("display_name is required")
	}
	if strings.TrimSpace(req.ChatAPIKey) == "" {
		return errors.New("chat_api_key is required")
	}
	if strings.TrimSpace(req.DefaultChatModel) == "" {
		return errors.New("default_chat_model is required")
	}
	return nil
}

func validateModelProviderUpdate(req updateModelProviderRequest) error {
	if req.DisplayName != nil && strings.TrimSpace(*req.DisplayName) == "" {
		return errors.New("display_name cannot be empty")
	}
	if req.ChatAPIKey != nil && strings.TrimSpace(*req.ChatAPIKey) == "" {
		return errors.New("chat_api_key cannot be empty")
	}
	if req.EmbeddingAPIKey != nil && *req.EmbeddingAPIKey != nil && strings.TrimSpace(**req.EmbeddingAPIKey) == "" {
		return errors.New("embedding_api_key cannot be empty")
	}
	if req.DefaultChatModel != nil && strings.TrimSpace(*req.DefaultChatModel) == "" {
		return errors.New("default_chat_model cannot be empty")
	}
	if req.Status != nil && *req.Status != "active" && *req.Status != "disabled" {
		return errors.New("status must be active or disabled")
	}
	return nil
}

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func normalizeOptionalStringPtr(value **string) **string {
	if value == nil {
		return nil
	}
	normalized := normalizeOptionalString(*value)
	return &normalized
}

func trimStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
