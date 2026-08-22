package contextmgr

import (
	"fmt"
	"strings"

	"freedinner/backend/internal/store"
)

func renderMemory(chunks []MemoryChunk, maxTokens int) (string, int, int) {
	if len(chunks) == 0 {
		return "", 0, 0
	}
	var builder strings.Builder
	builder.WriteString("Retrieved Memory\n")
	total := 0
	truncated := 0
	for _, chunk := range chunks {
		tokenCount := chunk.TokenCount
		if tokenCount <= 0 {
			tokenCount = EstimateTokens(chunk.Content)
		}
		if total+tokenCount > maxTokens && total > 0 {
			truncated++
			continue
		}
		total += tokenCount
		builder.WriteString(fmt.Sprintf("- [%s] %s\n", chunk.Layer, strings.TrimSpace(chunk.Content)))
	}
	return strings.TrimSpace(builder.String()), total, truncated
}

func renderSkills(sections []SkillSection, maxTokens int) (string, int, int) {
	if len(sections) == 0 {
		return "", 0, 0
	}
	var builder strings.Builder
	builder.WriteString("Procedural Skills\n")
	total := 0
	truncated := 0
	for _, section := range sections {
		tokenCount := section.TokenCount
		if tokenCount <= 0 {
			tokenCount = EstimateTokens(section.Content)
		}
		if total+tokenCount > maxTokens && total > 0 {
			truncated++
			continue
		}
		total += tokenCount
		builder.WriteString(fmt.Sprintf("- %s: %s\n", strings.TrimSpace(section.Title), strings.TrimSpace(section.Content)))
	}
	return strings.TrimSpace(builder.String()), total, truncated
}

func EstimateSkillTokens(sections []SkillSection) int {
	total := 0
	for _, section := range sections {
		if section.TokenCount > 0 {
			total += section.TokenCount
		} else {
			total += EstimateTokens(section.Content)
		}
	}
	return total
}

func renderSummaries(summaries []store.ConversationSummary) (string, int) {
	if len(summaries) == 0 {
		return "", 0
	}
	var builder strings.Builder
	builder.WriteString("Compressed Conversation Summary\n")
	total := 0
	for _, summary := range summaries {
		builder.WriteString(summary.Summary)
		builder.WriteString("\n")
		total += summary.TokenCount
	}
	return strings.TrimSpace(builder.String()), total
}
