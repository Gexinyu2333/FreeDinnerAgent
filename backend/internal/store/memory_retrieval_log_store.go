package store

import (
	"context"

	"github.com/google/uuid"
)

func (s *MemoryStore) CreateRetrievalLog(ctx context.Context, input MemoryRetrievalLogCreate) error {
	if input.LoadMode == "" {
		input.LoadMode = "standard"
	}
	_, err := s.db.Exec(ctx, `
		INSERT INTO memory_retrieval_logs (
			id, user_id, conversation_id, message_id, memory_layer, memory_ref_id, score, token_count, load_mode
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, uuid.NewString(), input.UserID, input.ConversationID, input.MessageID, input.MemoryLayer,
		input.MemoryRefID, input.Score, input.TokenCount, input.LoadMode)
	return err
}
