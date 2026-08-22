package memory

import (
	"context"
	"sort"
	"strings"

	"freedinner/backend/internal/store"
)

type RoutePlan struct {
	IncludeWorking  bool
	IncludeProfile  bool
	IncludeEpisodic bool
	IncludeSemantic bool
}

func (m *Manager) Retrieve(ctx context.Context, input RetrieveInput) (RetrieveResult, error) {
	input = normalizeRetrieveInput(input)
	plan := Route(input)
	var chunks []Chunk
	var semanticMode *string

	if plan.IncludeWorking && m.profiles != nil {
		memories, err := m.profiles.ListWorkingMemories(ctx, input.UserID, input.ConversationID, input.WorkingLimit)
		if err != nil {
			return RetrieveResult{}, err
		}
		for _, item := range memories {
			chunks = append(chunks, workingChunk(item))
		}
	}

	if plan.IncludeProfile && m.profiles != nil {
		memories, err := m.profiles.SearchProfileMemories(ctx, input.UserID, input.Query, input.ProfileLimit)
		if err != nil {
			return RetrieveResult{}, err
		}
		for _, item := range memories {
			chunks = append(chunks, profileChunk(item))
		}
	}

	if plan.IncludeEpisodic && m.profiles != nil {
		episodes, err := m.profiles.SearchEpisodes(ctx, input.UserID, input.Query, input.EpisodicLimit)
		if err != nil {
			return RetrieveResult{}, err
		}
		for _, item := range episodes {
			chunks = append(chunks, episodeChunk(item))
		}
	}

	if plan.IncludeSemantic && m.semantic != nil {
		result, err := m.semantic.SearchSemanticMemory(ctx, input.UserID, input.Query, input.SemanticLimit)
		if err != nil {
			return RetrieveResult{}, err
		}
		semanticMode = &result.Mode
		for _, item := range result.Chunks {
			chunks = append(chunks, semanticChunk(item))
		}
	}

	chunks = Compress(chunks, input.MaxMemoryTokens)
	if input.LogRetrieval && m.profiles != nil {
		for _, chunk := range chunks {
			_ = m.profiles.CreateRetrievalLog(ctx, store.MemoryRetrievalLogCreate{
				UserID:         input.UserID,
				ConversationID: input.ConversationID,
				MessageID:      input.MessageID,
				MemoryLayer:    chunk.Layer,
				MemoryRefID:    chunk.RefID,
				Score:          &chunk.Score,
				TokenCount:     chunk.TokenCount,
				LoadMode:       chunk.LoadMode,
			})
		}
	}

	return RetrieveResult{
		Chunks:       chunks,
		TokenCount:   sumTokens(chunks),
		UsedLayers:   usedLayers(chunks),
		Skipped:      skippedLayers(plan, chunks),
		SemanticMode: semanticMode,
	}, nil
}

func Route(input RetrieveInput) RoutePlan {
	query := strings.ToLower(input.Query)
	return RoutePlan{
		IncludeWorking:  input.IncludeWorking,
		IncludeProfile:  input.IncludeProfile,
		IncludeEpisodic: input.IncludeEpisodic || looksEpisodic(query),
		IncludeSemantic: input.IncludeSemantic || (input.SemanticOnDemand && looksSemantic(query)),
	}
}

func Compress(chunks []Chunk, maxTokens int) []Chunk {
	if maxTokens <= 0 {
		maxTokens = 1200
	}
	sort.SliceStable(chunks, func(i, j int) bool {
		if chunks[i].Score == chunks[j].Score {
			return layerPriority(chunks[i].Layer) < layerPriority(chunks[j].Layer)
		}
		return chunks[i].Score > chunks[j].Score
	})

	seen := map[string]bool{}
	result := make([]Chunk, 0, len(chunks))
	total := 0
	for _, chunk := range chunks {
		key := chunk.Layer + ":" + chunk.RefID
		if seen[key] {
			continue
		}
		seen[key] = true
		if chunk.TokenCount <= 0 {
			chunk.TokenCount = estimateTokens(chunk.Content)
		}
		if total+chunk.TokenCount > maxTokens && len(result) > 0 {
			continue
		}
		total += chunk.TokenCount
		result = append(result, chunk)
	}
	return result
}

func normalizeRetrieveInput(input RetrieveInput) RetrieveInput {
	if input.MaxMemoryTokens <= 0 {
		input.MaxMemoryTokens = 1200
	}
	if input.WorkingLimit <= 0 || input.WorkingLimit > 20 {
		input.WorkingLimit = 8
	}
	if input.ProfileLimit <= 0 || input.ProfileLimit > 20 {
		input.ProfileLimit = 8
	}
	if input.SemanticLimit <= 0 || input.SemanticLimit > 20 {
		input.SemanticLimit = 6
	}
	if input.EpisodicLimit <= 0 || input.EpisodicLimit > 20 {
		input.EpisodicLimit = 5
	}
	if !input.IncludeWorking && !input.IncludeProfile && !input.IncludeEpisodic && !input.IncludeSemantic {
		input.IncludeWorking = true
		input.IncludeProfile = true
		input.IncludeEpisodic = true
		input.SemanticOnDemand = true
	}
	return input
}

func looksSemantic(query string) bool {
	keywords := []string{"文档", "资料", "知识库", "搜索", "检索", "rag", "论文", "课程", "根据"}
	for _, keyword := range keywords {
		if strings.Contains(query, keyword) {
			return true
		}
	}
	return false
}

func looksEpisodic(query string) bool {
	keywords := []string{"上次", "之前", "以前", "类似", "经验", "怎么处理", "历史", "曾经", "again", "similar", "before"}
	for _, keyword := range keywords {
		if strings.Contains(query, keyword) {
			return true
		}
	}
	return false
}
