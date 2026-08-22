package contextmgr

import (
	"strings"
	"unicode"

	"freedinner/backend/internal/store"
)

func EstimateTokens(content string) int {
	if strings.TrimSpace(content) == "" {
		return 0
	}
	cjk := 0
	latin := 0
	other := 0
	for _, r := range content {
		switch {
		case unicode.Is(unicode.Han, r), unicode.Is(unicode.Hiragana, r), unicode.Is(unicode.Katakana, r), unicode.Is(unicode.Hangul, r):
			cjk++
		case r <= unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			latin++
		case unicode.IsSpace(r):
			continue
		default:
			other++
		}
	}
	return cjk + other/2 + latin/4 + 1
}

func normalizeBuildInput(input BuildInput) BuildInput {
	if input.MaxTokens <= 0 {
		input.MaxTokens = 12000
	}
	if input.RecentTurnLimit <= 0 {
		input.RecentTurnLimit = DefaultRecentTurnLimit
	}
	return input
}

func compressionStrategy(all []store.Message, selected []store.Message, maxTokens int, estimated int) *string {
	var strategies []string
	if len(selected) < len(all) {
		strategies = append(strategies, "recent_turn_limit")
	}
	if estimated > int(float64(maxTokens)*DefaultCompressionThresholdRate) {
		strategies = append(strategies, "token_threshold")
	}
	if len(strategies) == 0 {
		return nil
	}
	strategy := strings.Join(strategies, "+")
	return &strategy
}

func usedSections(memoryText, skillText, summaryText string) []string {
	sections := []string{"system", "recent_messages", "current_input"}
	if memoryText != "" {
		sections = append(sections, "memory")
	}
	if skillText != "" {
		sections = append(sections, "skills")
	}
	if summaryText != "" {
		sections = append(sections, "summary")
	}
	return sections
}
