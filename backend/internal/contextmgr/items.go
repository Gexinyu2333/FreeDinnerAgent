package contextmgr

import (
	"strings"

	"freedinner/backend/internal/store"
)

func buildItems(input BuildInput, summaries []store.ConversationSummary, messages []store.Message, report HealthReport, truncatedMemory int) []store.ContextBuildItemCreate {
	items := []store.ContextBuildItemCreate{
		{ItemType: "system", Title: stringPtr("Meta System Prompt + Safety"), TokenCount: report.SystemTokens, Priority: 100},
	}
	for _, chunk := range input.MemoryChunks {
		itemType := memoryItemType(chunk.Layer)
		refID := refIDPtr(chunk.RefID)
		title := stringPtr(chunk.Layer)
		items = append(items, store.ContextBuildItemCreate{ItemType: itemType, RefID: refID, Title: title, TokenCount: chunk.TokenCount, LoadMode: defaultLoadMode(chunk.LoadMode), WasTruncated: truncatedMemory > 0, Priority: 80})
	}
	for _, section := range input.SkillSections {
		refID := refIDPtr(section.RefID)
		title := stringPtr(section.Title)
		items = append(items, store.ContextBuildItemCreate{ItemType: "procedural_skill", RefID: refID, Title: title, TokenCount: section.TokenCount, LoadMode: defaultLoadMode(section.LoadMode), Priority: 75})
	}
	for _, summary := range summaries {
		refID := summary.ID
		items = append(items, store.ContextBuildItemCreate{ItemType: "summary", RefID: &refID, Title: stringPtr(summary.SummaryType), TokenCount: summary.TokenCount, LoadMode: "summary", WasCompressed: true, Priority: 60})
	}
	for index, message := range messages {
		refID := message.ID
		itemType := "recent_message"
		if index == len(messages)-1 && message.Role == "user" {
			itemType = "current_input"
		}
		items = append(items, store.ContextBuildItemCreate{ItemType: itemType, RefID: &refID, Title: stringPtr(message.Role), TokenCount: messageTokenCount(message), Priority: 90})
	}
	return items
}

func memoryItemType(layer string) string {
	switch layer {
	case "working":
		return "working_memory"
	case "profile":
		return "profile_memory"
	case "semantic":
		return "semantic_memory"
	case "episodic":
		return "episodic_memory"
	case "procedural":
		return "procedural_skill"
	default:
		return "profile_memory"
	}
}

func refIDPtr(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func stringPtr(value string) *string {
	return &value
}

func defaultLoadMode(value string) string {
	if value == "" {
		return "standard"
	}
	return value
}
