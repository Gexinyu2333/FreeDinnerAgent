package llm

import "strings"

func trimRunes(value string, max int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max]) + "..."
}

func refOrNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func stringPtr(value string) *string {
	return &value
}

func optionalStepID(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
