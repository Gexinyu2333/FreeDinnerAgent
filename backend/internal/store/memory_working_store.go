package store

import (
	"context"

	"github.com/google/uuid"
)

func (s *MemoryStore) UpsertWorkingMemory(ctx context.Context, input WorkingMemoryUpsert) (WorkingMemory, error) {
	if input.Category == "" {
		input.Category = "temporary_context"
	}
	if input.TokenCount <= 0 {
		input.TokenCount = estimateStoreTokens(input.MemoryValue)
	}
	return scanWorkingMemory(s.db.QueryRow(ctx, `
		INSERT INTO session_working_memories (
			id, user_id, conversation_id, memory_key, memory_value, category, token_count, expires_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (conversation_id, memory_key)
		DO UPDATE SET
			memory_value = EXCLUDED.memory_value,
			category = EXCLUDED.category,
			token_count = EXCLUDED.token_count,
			expires_at = EXCLUDED.expires_at,
			updated_at = NOW()
		RETURNING id, user_id, conversation_id, memory_key, memory_value, category, token_count, expires_at, created_at, updated_at
	`, uuid.NewString(), input.UserID, input.ConversationID, input.MemoryKey, input.MemoryValue,
		input.Category, input.TokenCount, input.ExpiresAt))
}

func (s *MemoryStore) ListWorkingMemories(ctx context.Context, userID, conversationID string, limit int) ([]WorkingMemory, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, conversation_id, memory_key, memory_value, category, token_count, expires_at, created_at, updated_at
		FROM session_working_memories
		WHERE user_id = $1
		  AND conversation_id = $2
		  AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY updated_at DESC
		LIMIT $3
	`, userID, conversationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	memories := make([]WorkingMemory, 0)
	for rows.Next() {
		memory, err := scanWorkingMemory(rows)
		if err != nil {
			return nil, err
		}
		memories = append(memories, memory)
	}
	return memories, rows.Err()
}
