package memory

import (
	"strings"

	"freedinner/backend/internal/store"
)

func workingChunk(item store.WorkingMemory) Chunk {
	return Chunk{
		Layer:      LayerWorking,
		RefID:      item.ID,
		Content:    item.MemoryKey + ": " + item.MemoryValue,
		Score:      1.0,
		TokenCount: item.TokenCount,
		Visibility: "private",
		Source:     item.Category,
		LoadMode:   "standard",
		Metadata: Metadata{
			"memory_key": item.MemoryKey,
			"category":   item.Category,
		},
	}
}

func profileChunk(item store.ProfileMemory) Chunk {
	score := (float64(item.Importance) / 5.0) * item.Confidence
	return Chunk{
		Layer:      LayerProfile,
		RefID:      item.ID,
		Content:    item.Title + ": " + item.Content,
		Score:      score,
		TokenCount: estimateTokens(item.Title + item.Content),
		Visibility: "private",
		Source:     item.MemoryType,
		LoadMode:   "standard",
		Metadata: Metadata{
			"memory_type": item.MemoryType,
			"scope":       item.Scope,
			"importance":  item.Importance,
			"confidence":  item.Confidence,
		},
	}
}

func semanticChunk(item SemanticChunk) Chunk {
	score := 0.5
	if item.Similarity != nil {
		score = *item.Similarity
	}
	source := "knowledge"
	if item.DocumentTitle != nil && strings.TrimSpace(*item.DocumentTitle) != "" {
		source = *item.DocumentTitle
	}
	return Chunk{
		Layer:      LayerSemantic,
		RefID:      item.ID,
		Content:    item.Content,
		Score:      score,
		TokenCount: item.TokenCount,
		Visibility: item.Visibility,
		Source:     source,
		LoadMode:   "standard",
		Metadata: Metadata{
			"document_id":   item.DocumentID,
			"chunk_index":   item.ChunkIndex,
			"has_embedding": item.HasEmbedding,
		},
	}
}

func episodeChunk(item store.EpisodeMatch) Chunk {
	content := "历史经历：" + item.AgentSummary
	if strings.TrimSpace(item.FinalResponse) != "" {
		content += "\n当时回复：" + trimText(item.FinalResponse, 220)
	}
	metadata := Metadata{
		"status":     item.Status,
		"importance": item.Importance,
		"tags":       item.Tags,
	}
	if item.TaskType != nil {
		metadata["task_type"] = *item.TaskType
	}
	return Chunk{
		Layer:      LayerEpisodic,
		RefID:      item.ID,
		Content:    content,
		Score:      item.Score,
		TokenCount: estimateTokens(content),
		Visibility: "private",
		Source:     "episode",
		LoadMode:   "light",
		Metadata:   metadata,
	}
}
