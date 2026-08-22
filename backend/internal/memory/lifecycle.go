package memory

import (
	"context"

	"freedinner/backend/internal/store"
)

func (m *Manager) UpsertWorkingMemory(ctx context.Context, input WorkingMemoryInput) (store.WorkingMemory, error) {
	if m.profiles == nil {
		return store.WorkingMemory{}, nil
	}
	return m.profiles.UpsertWorkingMemory(ctx, store.WorkingMemoryUpsert{
		UserID:         input.UserID,
		ConversationID: input.ConversationID,
		MemoryKey:      input.MemoryKey,
		MemoryValue:    input.MemoryValue,
		Category:       input.Category,
	})
}

func (m *Manager) CreateEpisode(ctx context.Context, input EpisodeInput) (store.Episode, error) {
	if m.profiles == nil {
		return store.Episode{}, nil
	}
	return m.profiles.CreateEpisode(ctx, store.EpisodeCreate{
		UserID:             input.UserID,
		ConversationID:     input.ConversationID,
		UserMessageID:      input.UserMessageID,
		AssistantMessageID: input.AssistantMessageID,
		UserInput:          input.UserInput,
		AgentSummary:       input.AgentSummary,
		FinalResponse:      input.FinalResponse,
		TaskType:           input.TaskType,
		Status:             input.Status,
		Tags:               input.Tags,
	})
}
