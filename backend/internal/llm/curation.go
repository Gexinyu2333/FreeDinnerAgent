package llm

import (
	"context"
	"encoding/json"
	"strings"

	"freedinner/backend/internal/memory"
	"freedinner/backend/internal/store"
)

func (s *Service) curateEpisode(ctx context.Context, in loopInput, assistantMessage store.Message, status string) error {
	if s.memories == nil {
		return nil
	}
	summary := strings.TrimSpace(in.userMessage.Content)
	if len([]rune(summary)) > 120 {
		summary = string([]rune(summary)[:120]) + "..."
	}
	_, err := s.memories.UpsertWorkingMemory(ctx, memory.WorkingMemoryInput{
		UserID:         in.userID,
		ConversationID: in.conversationID,
		MemoryKey:      "last_episode",
		MemoryValue:    "用户：" + summary + "\n助手：" + trimRunes(assistantMessage.Content, 160),
		Category:       "temporary_context",
	})
	if err != nil {
		return err
	}
	taskType := inferTaskType(in.userMessage.Content)
	episode, err := s.memories.CreateEpisode(ctx, memory.EpisodeInput{
		UserID:             in.userID,
		ConversationID:     in.conversationID,
		UserMessageID:      &in.userMessage.ID,
		AssistantMessageID: &assistantMessage.ID,
		UserInput:          in.userMessage.Content,
		AgentSummary:       "用户请求：" + summary + "；助手已生成回复。",
		FinalResponse:      assistantMessage.Content,
		TaskType:           taskType,
		Status:             status,
		Tags:               episodeTags(in.userMessage.Content),
	})
	if err != nil {
		return err
	}
	payload, _ := json.Marshal(map[string]any{
		"episode_id":      episode.ID,
		"conversation_id": in.conversationID,
		"task_type":       taskType,
	})
	_, _ = s.memories.EnqueueCuratorJob(ctx, in.userID, "episode_summary", payload)
	if in.cfg.DreamingEnabled && shouldTriggerDreaming(in.userMessage.Content) {
		_, _ = s.memories.EnqueueCuratorJob(ctx, in.userID, "dreaming", payload)
		_, _ = s.memories.DistillSkillFromEpisode(ctx, memory.SkillDistillationInput{
			UserID:   in.userID,
			Episode:  episode,
			Query:    in.userMessage.Content,
			Response: assistantMessage.Content,
			TaskType: taskType,
		})
	}
	return nil
}

func inferTaskType(content string) *string {
	lowered := strings.ToLower(content)
	taskType := "chat"
	switch {
	case strings.Contains(lowered, "任务") || strings.Contains(lowered, "待办") || strings.Contains(lowered, "提醒"):
		taskType = "task"
	case strings.Contains(lowered, "总结") || strings.Contains(lowered, "摘要"):
		taskType = "summary"
	case strings.Contains(lowered, "搜索") || strings.Contains(lowered, "检索") || strings.Contains(lowered, "资料"):
		taskType = "search"
	case strings.Contains(lowered, "代码") || strings.Contains(lowered, "运行"):
		taskType = "workspace"
	}
	return &taskType
}

func episodeTags(content string) []string {
	tags := []string{}
	if taskType := inferTaskType(content); taskType != nil {
		tags = append(tags, *taskType)
	}
	if shouldTriggerDreaming(content) {
		tags = append(tags, "dreaming_candidate")
	}
	return tags
}

func shouldTriggerDreaming(content string) bool {
	lowered := strings.ToLower(content)
	keywords := []string{"反复", "每次", "以后", "流程", "习惯", "沉淀", "skill", "技能"}
	for _, keyword := range keywords {
		if strings.Contains(lowered, keyword) {
			return true
		}
	}
	return false
}
