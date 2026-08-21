package tool

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
)

var ErrInvalidArguments = errors.New("tool arguments failed schema validation")

func validateArguments(schemaRaw json.RawMessage, argumentsRaw json.RawMessage) (json.RawMessage, error) {
	if len(argumentsRaw) == 0 {
		argumentsRaw = json.RawMessage(`{}`)
	}
	var schema map[string]any
	if err := json.Unmarshal(schemaRaw, &schema); err != nil {
		return nil, fmt.Errorf("invalid parameter schema: %w", err)
	}
	var arguments map[string]any
	if err := json.Unmarshal(argumentsRaw, &arguments); err != nil {
		return nil, fmt.Errorf("%w: arguments must be a JSON object", ErrInvalidArguments)
	}
	if err := validateObjectSchema(schema, arguments); err != nil {
		return nil, err
	}
	normalized, err := json.Marshal(arguments)
	if err != nil {
		return nil, err
	}
	return normalized, nil
}

func validateObjectSchema(schema map[string]any, arguments map[string]any) error {
	if schemaType, _ := schema["type"].(string); schemaType != "" && schemaType != "object" {
		return fmt.Errorf("unsupported root schema type %q", schemaType)
	}
	required := stringSlice(schema["required"])
	for _, name := range required {
		value, ok := arguments[name]
		if !ok || value == nil {
			return fmt.Errorf("%w: %s is required", ErrInvalidArguments, name)
		}
	}
	properties, _ := schema["properties"].(map[string]any)
	for name, value := range arguments {
		propertySchema, ok := properties[name].(map[string]any)
		if !ok {
			continue
		}
		if err := validateValue(name, value, propertySchema); err != nil {
			return err
		}
	}
	return nil
}

func validateValue(name string, value any, schema map[string]any) error {
	expectedType, _ := schema["type"].(string)
	if expectedType != "" && !matchesJSONType(value, expectedType) {
		return fmt.Errorf("%w: %s must be %s", ErrInvalidArguments, name, expectedType)
	}
	if enumValues := stringSlice(schema["enum"]); len(enumValues) > 0 {
		stringValue, ok := value.(string)
		if !ok || !containsString(enumValues, stringValue) {
			return fmt.Errorf("%w: %s must be one of %s", ErrInvalidArguments, name, strings.Join(enumValues, ", "))
		}
	}
	if minimum, ok := numberValue(schema["minimum"]); ok {
		valueNumber, valueOK := numberValue(value)
		if !valueOK || valueNumber < minimum {
			return fmt.Errorf("%w: %s must be >= %s", ErrInvalidArguments, name, formatNumber(minimum))
		}
	}
	if maximum, ok := numberValue(schema["maximum"]); ok {
		valueNumber, valueOK := numberValue(value)
		if !valueOK || valueNumber > maximum {
			return fmt.Errorf("%w: %s must be <= %s", ErrInvalidArguments, name, formatNumber(maximum))
		}
	}
	if expectedType == "array" {
		items, _ := schema["items"].(map[string]any)
		values, _ := value.([]any)
		for index, item := range values {
			if err := validateValue(fmt.Sprintf("%s[%d]", name, index), item, items); err != nil {
				return err
			}
		}
	}
	return nil
}

func matchesJSONType(value any, expected string) bool {
	switch expected {
	case "string":
		_, ok := value.(string)
		return ok
	case "integer":
		number, ok := numberValue(value)
		return ok && math.Trunc(number) == number
	case "number":
		_, ok := numberValue(value)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "array":
		_, ok := value.([]any)
		return ok
	case "object":
		_, ok := value.(map[string]any)
		return ok
	default:
		return true
	}
}

func numberValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil
	default:
		return 0, false
	}
}

func stringSlice(value any) []string {
	values, ok := value.([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, item := range values {
		if text, ok := item.(string); ok {
			result = append(result, text)
		}
	}
	return result
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func formatNumber(value float64) string {
	if math.Trunc(value) == value {
		return fmt.Sprintf("%.0f", value)
	}
	return fmt.Sprintf("%g", value)
}
