package market

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"freedinner/backend/internal/store"
)

var ErrInvalidInput = errors.New("invalid market input")

type Service struct {
	market      *store.MarketStore
	agentConfig *store.AgentConfigStore
}

type PromptVariableInput struct {
	Name          string
	DisplayName   string
	Description   *string
	DefaultValue  *string
	Required      bool
	ValueType     string
	AllowedValues []string
}

type CreatePromptTemplateInput struct {
	UserID      string
	Name        string
	DisplayName string
	Description string
	Category    string
	Tags        []string
	Visibility  string
	Content     string
	ChangeNote  *string
	Variables   []PromptVariableInput
}

type ForkPromptTemplateInput struct {
	UserID      string
	VersionID   string
	Name        string
	DisplayName string
	Description string
}

type CreatePromptTemplateResult struct {
	Template store.SystemPromptTemplate        `json:"template"`
	Version  store.SystemPromptTemplateVersion `json:"version"`
	Item     store.MarketplaceItem             `json:"marketplace_item"`
}

type PreviewPromptInput struct {
	UserID    string
	VersionID string
	Variables map[string]string
	Override  *string
}

type PromptPreview struct {
	Template  store.SystemPromptTemplate        `json:"template"`
	Version   store.SystemPromptTemplateVersion `json:"version"`
	Variables []store.SystemPromptVariable      `json:"variables"`
	Content   string                            `json:"content"`
	Tokens    int                               `json:"tokens"`
}

type RateItemResult struct {
	Review store.MarketplaceReview `json:"review"`
	Item   store.MarketplaceItem   `json:"marketplace_item"`
}

func NewService(marketStore *store.MarketStore, agentConfig *store.AgentConfigStore) *Service {
	return &Service{market: marketStore, agentConfig: agentConfig}
}

func (s *Service) ListItems(ctx context.Context, userID string, itemType *string, installedOnly bool, limit int) ([]store.MarketplaceItemView, error) {
	return s.market.ListMarketplaceItemViews(ctx, userID, itemType, installedOnly, limit)
}

func (s *Service) Install(ctx context.Context, userID, itemID string) (store.CapabilityInstall, error) {
	return s.market.InstallCapability(ctx, userID, itemID)
}

func (s *Service) RateItem(ctx context.Context, userID, itemID string, rating int, comment *string) (RateItemResult, error) {
	if rating < 1 || rating > 5 {
		return RateItemResult{}, fmt.Errorf("%w: rating must be between 1 and 5", ErrInvalidInput)
	}
	if comment != nil {
		trimmed := strings.TrimSpace(*comment)
		if trimmed == "" {
			comment = nil
		} else {
			comment = &trimmed
		}
	}
	review, item, err := s.market.RateMarketplaceItem(ctx, userID, itemID, rating, comment)
	if err != nil {
		return RateItemResult{}, err
	}
	return RateItemResult{Review: review, Item: item}, nil
}

func (s *Service) SetInstallEnabled(ctx context.Context, userID, installID string, enabled bool) (store.CapabilityInstall, error) {
	return s.market.SetInstallEnabled(ctx, userID, installID, enabled)
}

func (s *Service) Bind(ctx context.Context, userID, agentConfigID, capabilityType, capabilityRefID, loadMode string, priority int) (store.AgentCapabilityBinding, error) {
	if agentConfigID == "" {
		cfg, err := s.agentConfig.GetDefault(ctx, userID)
		if err != nil {
			return store.AgentCapabilityBinding{}, err
		}
		agentConfigID = cfg.ID
	}
	if capabilityType == "system_prompt_template" {
		if _, _, err := s.market.FindSystemPromptVersion(ctx, userID, capabilityRefID); err != nil {
			return store.AgentCapabilityBinding{}, err
		}
	}
	return s.market.BindCapability(ctx, userID, agentConfigID, capabilityType, capabilityRefID, normalizeLoadMode(loadMode), priority)
}

func (s *Service) SetBindingEnabled(ctx context.Context, userID, bindingID string, enabled bool) (store.AgentCapabilityBinding, error) {
	return s.market.SetBindingEnabled(ctx, userID, bindingID, enabled)
}

func (s *Service) CreatePromptTemplate(ctx context.Context, input CreatePromptTemplateInput) (CreatePromptTemplateResult, error) {
	content := strings.TrimSpace(input.Content)
	if content == "" {
		return CreatePromptTemplateResult{}, fmt.Errorf("%w: content is required", ErrInvalidInput)
	}
	safetyPolicy, err := PromptTemplateSafetyPolicy(content)
	if err != nil {
		return CreatePromptTemplateResult{}, err
	}
	variables, err := normalizePromptVariables(content, input.Variables)
	if err != nil {
		return CreatePromptTemplateResult{}, err
	}
	template, version, item, err := s.market.CreateSystemPromptTemplate(ctx, store.CreateSystemPromptTemplateInput{
		UserID:       input.UserID,
		Name:         strings.TrimSpace(input.Name),
		DisplayName:  strings.TrimSpace(input.DisplayName),
		Description:  strings.TrimSpace(input.Description),
		Category:     strings.TrimSpace(input.Category),
		Tags:         compactStrings(input.Tags),
		Visibility:   normalizeVisibility(input.Visibility),
		Content:      content,
		ChangeNote:   input.ChangeNote,
		SafetyPolicy: safetyPolicy,
		Variables:    variables,
	})
	if err != nil {
		return CreatePromptTemplateResult{}, err
	}
	return CreatePromptTemplateResult{Template: template, Version: version, Item: item}, nil
}

func (s *Service) ForkPromptTemplate(ctx context.Context, input ForkPromptTemplateInput) (CreatePromptTemplateResult, error) {
	template, version, err := s.market.FindSystemPromptVersion(ctx, input.UserID, input.VersionID)
	if err != nil {
		return CreatePromptTemplateResult{}, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = template.Name + "_fork"
	}
	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		displayName = template.DisplayName + " Fork"
	}
	description := strings.TrimSpace(input.Description)
	if description == "" {
		description = "Forked from " + template.DisplayName
	}
	variables, err := s.market.ListSystemPromptVariables(ctx, version.ID)
	if err != nil {
		return CreatePromptTemplateResult{}, err
	}
	forkVariables := make([]PromptVariableInput, 0, len(variables))
	for _, variable := range variables {
		forkVariables = append(forkVariables, PromptVariableInput{
			Name:         variable.Name,
			DisplayName:  variable.DisplayName,
			Description:  variable.Description,
			DefaultValue: variable.DefaultValue,
			Required:     variable.Required,
			ValueType:    variable.ValueType,
		})
	}
	return s.CreatePromptTemplate(ctx, CreatePromptTemplateInput{
		UserID:      input.UserID,
		Name:        name,
		DisplayName: displayName,
		Description: description,
		Category:    template.Category,
		Tags:        template.Tags,
		Visibility:  "private",
		Content:     version.Content,
		Variables:   forkVariables,
	})
}

func (s *Service) PreviewPrompt(ctx context.Context, input PreviewPromptInput) (PromptPreview, error) {
	template, version, err := s.market.FindSystemPromptVersion(ctx, input.UserID, input.VersionID)
	if err != nil {
		return PromptPreview{}, err
	}
	variables, err := s.market.ListSystemPromptVariables(ctx, version.ID)
	if err != nil {
		return PromptPreview{}, err
	}
	resolved, err := ResolvePromptVariables(variables, input.Variables)
	if err != nil {
		return PromptPreview{}, err
	}
	content := RenderPrompt(version.Content, resolved)
	if input.Override != nil && strings.TrimSpace(*input.Override) != "" {
		content = content + "\n\nUser Custom Override:\n" + strings.TrimSpace(*input.Override)
	}
	return PromptPreview{Template: template, Version: version, Variables: variables, Content: content, Tokens: EstimateTokens(content)}, nil
}

func normalizeLoadMode(value string) string {
	switch strings.TrimSpace(value) {
	case "light", "standard", "full":
		return strings.TrimSpace(value)
	default:
		return "auto"
	}
}

func normalizeVisibility(value string) string {
	if strings.TrimSpace(value) == "public" {
		return "public"
	}
	return "private"
}
