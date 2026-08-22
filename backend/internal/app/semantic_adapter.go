package app

import (
	"context"

	"freedinner/backend/internal/knowledge"
	memorysvc "freedinner/backend/internal/memory"
)

type semanticMemoryAdapter struct {
	knowledge *knowledge.Service
}

func (a semanticMemoryAdapter) SearchSemanticMemory(ctx context.Context, userID, query string, limit int) (memorysvc.SemanticSearchResult, error) {
	result, err := a.knowledge.Search(ctx, userID, query, limit)
	if err != nil {
		return memorysvc.SemanticSearchResult{}, err
	}
	chunks := make([]memorysvc.SemanticChunk, 0, len(result.Chunks))
	for _, chunk := range result.Chunks {
		chunks = append(chunks, memorysvc.SemanticChunk{
			ID:            chunk.ID,
			DocumentID:    chunk.DocumentID,
			Visibility:    chunk.Visibility,
			ChunkIndex:    chunk.ChunkIndex,
			Content:       chunk.Content,
			TokenCount:    chunk.TokenCount,
			Similarity:    chunk.Similarity,
			DocumentTitle: chunk.DocumentTitle,
			HasEmbedding:  chunk.HasEmbedding,
		})
	}
	return memorysvc.SemanticSearchResult{Mode: result.Mode, Chunks: chunks}, nil
}
