package store

import "strings"

func normalizeVisibility(value string) string {
	if strings.TrimSpace(value) == "public" {
		return "public"
	}
	return "private"
}

func estimateTokens(content string) int {
	runes := len([]rune(content))
	if runes == 0 {
		return 0
	}
	return runes/3 + 1
}
