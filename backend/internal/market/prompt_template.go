package market

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"freedinner/backend/internal/store"
)

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

func normalizeValueType(value string) string {
	switch strings.TrimSpace(value) {
	case "number", "boolean", "enum", "json":
		return strings.TrimSpace(value)
	default:
		return "string"
	}
}

func ScanPromptTemplateSafety(content string) error {
	_, err := PromptTemplateSafetyPolicy(content)
	return err
}

func PromptTemplateSafetyPolicy(content string) (json.RawMessage, error) {
	lowered := strings.ToLower(content)
	dangerous := []string{
		"ignore previous instructions",
		"忽略以上",
		"忽略之前",
		"泄露 api key",
		"输出 api key",
		"reveal api key",
		"bypass approval",
		"绕过审批",
		"无需审批执行",
		"读取所有用户",
		"跨用户",
	}
	for _, keyword := range dangerous {
		if strings.Contains(lowered, keyword) {
			return nil, fmt.Errorf("%w: system prompt template contains unsafe instruction %q", ErrInvalidInput, keyword)
		}
	}
	raw, err := json.Marshal(map[string]any{
		"scan":          "rule_based",
		"review_status": "auto_approved",
		"matched":       []string{},
	})
	if err != nil {
		return nil, err
	}
	return json.RawMessage(raw), nil
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
