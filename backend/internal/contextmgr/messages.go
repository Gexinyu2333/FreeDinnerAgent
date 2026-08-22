package contextmgr

import (
	"fmt"
	"strings"

	"freedinner/backend/internal/store"
)

func SelectRecentMessages(messages []store.Message, recentTurnLimit int) ([]store.Message, int) {
	if recentTurnLimit <= 0 {
		recentTurnLimit = DefaultRecentTurnLimit
	}
	userTurnIndexes := make([]int, 0)
	for index, message := range messages {
		if message.Role == "user" {
			userTurnIndexes = append(userTurnIndexes, index)
		}
	}
	if len(userTurnIndexes) <= recentTurnLimit {
		return messages, 0
	}
	start := userTurnIndexes[len(userTurnIndexes)-recentTurnLimit]
	return messages[start:], len(userTurnIndexes) - recentTurnLimit
}

func SummarizeMessages(messages []store.Message) string {
	var builder strings.Builder
	builder.WriteString("Conversation Summary\n")
	builder.WriteString("- 用户目标：\n")
	builder.WriteString("- 已确认约束：\n")
	builder.WriteString("- 关键事实：\n")
	builder.WriteString("- 已执行工具：\n")
	builder.WriteString("- 工具结果：\n")
	builder.WriteString("- 未完成事项：\n")
	builder.WriteString("- 冲突或待确认点：\n")
	for _, message := range messages {
		if !isPromptRole(message.Role) {
			continue
		}
		content := strings.TrimSpace(message.Content)
		if content == "" {
			continue
		}
		if len([]rune(content)) > 180 {
			content = string([]rune(content)[:180]) + "..."
		}
		builder.WriteString(fmt.Sprintf("- %s: %s\n", message.Role, content))
	}
	return builder.String()
}

func messageTokenCount(message store.Message) int {
	if message.TokenCount > 0 {
		return message.TokenCount
	}
	return EstimateTokens(message.Content)
}

func countUserTurns(messages []store.Message) int {
	count := 0
	for _, message := range messages {
		if message.Role == "user" {
			count++
		}
	}
	return count
}

func isPromptRole(role string) bool {
	return role == "user" || role == "assistant" || role == "system"
}
