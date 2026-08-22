package memory

import (
	"strings"
	"unicode"
)

func trimText(value string, max int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max]) + "..."
}

func shortID(value string) string {
	if len(value) <= 8 {
		return value
	}
	return value[:8]
}

func layerPriority(layer string) int {
	switch layer {
	case LayerWorking:
		return 0
	case LayerProfile:
		return 1
	case LayerProcedural:
		return 2
	case LayerEpisodic:
		return 3
	case LayerSemantic:
		return 4
	default:
		return 9
	}
}

func estimateTokens(content string) int {
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

func stringPtr(value string) *string {
	return &value
}

func sumTokens(chunks []Chunk) int {
	total := 0
	for _, chunk := range chunks {
		total += chunk.TokenCount
	}
	return total
}

func usedLayers(chunks []Chunk) []string {
	seen := map[string]bool{}
	var layers []string
	for _, chunk := range chunks {
		if !seen[chunk.Layer] {
			seen[chunk.Layer] = true
			layers = append(layers, chunk.Layer)
		}
	}
	return layers
}

func skippedLayers(plan RoutePlan, chunks []Chunk) []string {
	had := map[string]bool{}
	for _, chunk := range chunks {
		had[chunk.Layer] = true
	}
	var skipped []string
	if plan.IncludeWorking && !had[LayerWorking] {
		skipped = append(skipped, LayerWorking)
	}
	if plan.IncludeProfile && !had[LayerProfile] {
		skipped = append(skipped, LayerProfile)
	}
	if plan.IncludeSemantic && !had[LayerSemantic] {
		skipped = append(skipped, LayerSemantic)
	}
	return skipped
}
