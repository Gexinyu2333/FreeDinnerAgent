package market

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
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

func (s *Service) ListItems(ctx context.Context, userID string, itemType *string, installedOnly bool, limit int) ([]store.MarketplaceItem, error) {
	return s.market.ListMarketplaceItems(ctx, userID, itemType, installedOnly, limit)
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
	variables, err := normalizePromptVariables(content, input.Variables)
	if err != nil {
		return CreatePromptTemplateResult{}, err
	}
	template, version, item, err := s.market.CreateSystemPromptTemplate(ctx, store.CreateSystemPromptTemplateInput{
		UserID:      input.UserID,
		Name:        strings.TrimSpace(input.Name),
		DisplayName: strings.TrimSpace(input.DisplayName),
		Description: strings.TrimSpace(input.Description),
		Category:    strings.TrimSpace(input.Category),
		Tags:        compactStrings(input.Tags),
		Visibility:  normalizeVisibility(input.Visibility),
		Content:     content,
		ChangeNote:  input.ChangeNote,
		Variables:   variables,
	})
	if err != nil {
		return CreatePromptTemplateResult{}, err
	}
	return CreatePromptTemplateResult{Template: template, Version: version, Item: item}, nil
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

func RenderPrompt(content string, variables map[string]string) string {
	result := content
	for key, value := range variables {
		result = strings.ReplaceAll(result, "{"+key+"}", value)
	}
	return result
}

func ResolvePromptVariables(definitions []store.SystemPromptVariable, values map[string]string) (map[string]string, error) {
	if values == nil {
		values = map[string]string{}
	}
	resolved := map[string]string{}
	for _, definition := range definitions {
		value, ok := values[definition.Name]
		value = strings.TrimSpace(value)
		if !ok || value == "" {
			if definition.DefaultValue != nil {
				value = strings.TrimSpace(*definition.DefaultValue)
			}
		}
		if definition.Required && value == "" {
			return nil, fmt.Errorf("%w: missing required variable %s", ErrInvalidInput, definition.Name)
		}
		if value != "" {
			if err := validateVariableValue(definition, value); err != nil {
				return nil, err
			}
		}
		resolved[definition.Name] = value
	}
	return resolved, nil
}

var promptVariablePattern = regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*)\}`)

func ExtractPromptVariables(content string) []string {
	matches := promptVariablePattern.FindAllStringSubmatch(content, -1)
	seen := map[string]bool{}
	var names []string
	for _, match := range matches {
		if len(match) < 2 || seen[match[1]] {
			continue
		}
		seen[match[1]] = true
		names = append(names, match[1])
	}
	sort.Strings(names)
	return names
}

func normalizePromptVariables(content string, inputs []PromptVariableInput) ([]store.SystemPromptVariableInput, error) {
	byName := map[string]PromptVariableInput{}
	for _, input := range inputs {
		name := strings.TrimSpace(input.Name)
		if name == "" {
			return nil, fmt.Errorf("%w: variable name is required", ErrInvalidInput)
		}
		if !promptVariablePattern.MatchString("{" + name + "}") {
			return nil, fmt.Errorf("%w: invalid variable name %s", ErrInvalidInput, name)
		}
		input.Name = name
		byName[name] = input
	}
	for _, name := range ExtractPromptVariables(content) {
		if _, ok := byName[name]; !ok {
			byName[name] = PromptVariableInput{Name: name, DisplayName: name, ValueType: "string"}
		}
	}
	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)
	result := make([]store.SystemPromptVariableInput, 0, len(names))
	for _, name := range names {
		input := byName[name]
		valueType := normalizeValueType(input.ValueType)
		metadata, err := variableMetadata(input.AllowedValues)
		if err != nil {
			return nil, err
		}
		displayName := strings.TrimSpace(input.DisplayName)
		if displayName == "" {
			displayName = name
		}
		result = append(result, store.SystemPromptVariableInput{
			Name:         name,
			DisplayName:  displayName,
			Description:  trimStringPtr(input.Description),
			DefaultValue: trimStringPtr(input.DefaultValue),
			Required:     input.Required,
			ValueType:    valueType,
			Metadata:     metadata,
		})
	}
	return result, nil
}

func validateVariableValue(definition store.SystemPromptVariable, value string) error {
	switch definition.ValueType {
	case "number":
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			return fmt.Errorf("%w: variable %s must be number", ErrInvalidInput, definition.Name)
		}
	case "boolean":
		if _, err := strconv.ParseBool(value); err != nil {
			return fmt.Errorf("%w: variable %s must be boolean", ErrInvalidInput, definition.Name)
		}
	case "json":
		var raw any
		if err := json.Unmarshal([]byte(value), &raw); err != nil {
			return fmt.Errorf("%w: variable %s must be json", ErrInvalidInput, definition.Name)
		}
	case "enum":
		allowed := allowedValues(definition.Metadata)
		if len(allowed) == 0 {
			return nil
		}
		for _, candidate := range allowed {
			if value == candidate {
				return nil
			}
		}
		return fmt.Errorf("%w: variable %s must be one of %s", ErrInvalidInput, definition.Name, strings.Join(allowed, ", "))
	}
	return nil
}

func EstimateTokens(content string) int {
	runes := len([]rune(content))
	if runes == 0 {
		return 0
	}
	return runes/3 + 1
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

func normalizeValueType(value string) string {
	switch strings.TrimSpace(value) {
	case "number", "boolean", "enum", "json":
		return strings.TrimSpace(value)
	default:
		return "string"
	}
}

func variableMetadata(allowed []string) (json.RawMessage, error) {
	allowed = compactStrings(allowed)
	if len(allowed) == 0 {
		return json.RawMessage(`{}`), nil
	}
	raw, err := json.Marshal(map[string][]string{"allowed_values": allowed})
	if err != nil {
		return nil, err
	}
	return json.RawMessage(raw), nil
}

func allowedValues(raw json.RawMessage) []string {
	var metadata struct {
		AllowedValues []string `json:"allowed_values"`
	}
	if len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return nil
	}
	return compactStrings(metadata.AllowedValues)
}

func trimStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func compactStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func JSONVariables(raw json.RawMessage) map[string]string {
	result := map[string]string{}
	if len(raw) == 0 {
		return result
	}
	_ = json.Unmarshal(raw, &result)
	return result
}
