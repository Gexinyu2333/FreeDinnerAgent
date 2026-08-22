package store

import "strings"

func normalizedTags(tags []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		result = append(result, tag)
	}
	return result
}

func estimateStoreTokens(content string) int {
	runes := len([]rune(content))
	if runes == 0 {
		return 0
	}
	return runes/4 + 1
}

func stringPtr(value string) *string {
	return &value
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
