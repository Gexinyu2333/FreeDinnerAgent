package memory

import (
	"context"
	"sort"
	"strings"

	"freedinner/backend/internal/store"
)

const (
	LayerWorking    = "working"
	LayerProfile    = "profile"
	LayerEpisodic   = "episodic"
	LayerProcedural = "procedural"
	LayerSemantic   = "semantic"
)

type ProfileRetriever interface {
	ListWorkingMemories(ctx context.Context, userID, conversationID string, limit int) ([]store.WorkingMemory, error)
	UpsertWorkingMemory(ctx context.Context, input store.WorkingMemoryUpsert) (store.WorkingMemory, error)
	SearchProfileMemories(ctx context.Context, userID, query string, limit int) ([]store.ProfileMemory, error)
	CreateRetrievalLog(ctx context.Context, input store.MemoryRetrievalLogCreate) error
	CreateEpisode(ctx context.Context, input store.EpisodeCreate) (store.Episode, error)
	MatchSkillDisclosures(ctx context.Context, userID, query, loadMode string, limit int) ([]store.SkillDisclosure, error)
	CreateCuratorJob(ctx context.Context, input store.CuratorJobCreate) (store.CuratorJob, error)
	CreateDreamingSession(ctx context.Context, input store.DreamingSessionCreate) (store.DreamingSession, error)
	FinishDreamingSession(ctx context.Context, sessionID, userID, status string, outputSummary *string) (store.DreamingSession, error)
	CreateDreamingInsight(ctx context.Context, input store.DreamingInsightCreate) (store.DreamingInsight, error)
}

type SemanticRetriever interface {
	SearchSemanticMemory(ctx context.Context, userID, query string, limit int) (SemanticSearchResult, error)
}

type Manager struct {
	profiles ProfileRetriever
	semantic SemanticRetriever
}

type RetrieveInput struct {
	UserID           string
	ConversationID   string
	MessageID        *string
	Query            string
	MaxMemoryTokens  int
	ProfileLimit     int
	WorkingLimit     int
	SemanticLimit    int
	IncludeWorking   bool
	IncludeProfile   bool
	IncludeSemantic  bool
	LogRetrieval     bool
	SemanticOnDemand bool
}

type RetrieveResult struct {
	Chunks       []Chunk  `json:"chunks"`
	TokenCount   int      `json:"token_count"`
	UsedLayers   []string `json:"used_layers"`
	Skipped      []string `json:"skipped"`
	SemanticMode *string  `json:"semantic_mode,omitempty"`
}

type Chunk struct {
	Layer      string   `json:"layer"`
	RefID      string   `json:"ref_id"`
	Content    string   `json:"content"`
	Score      float64  `json:"score"`
	TokenCount int      `json:"token_count"`
	Visibility string   `json:"visibility"`
	Source     string   `json:"source"`
	LoadMode   string   `json:"load_mode"`
	Metadata   Metadata `json:"metadata"`
}

type Metadata map[string]any

type SemanticSearchResult struct {
	Mode   string
	Chunks []SemanticChunk
}

type SemanticChunk struct {
	ID            string
	DocumentID    string
	Visibility    string
	ChunkIndex    int
	Content       string
	TokenCount    int
	Similarity    *float64
	DocumentTitle *string
	HasEmbedding  bool
}

type WorkingMemoryInput struct {
	UserID         string
	ConversationID string
	MemoryKey      string
	MemoryValue    string
	Category       string
}

type EpisodeInput struct {
	UserID             string
	ConversationID     string
	UserMessageID      *string
	AssistantMessageID *string
	UserInput          string
	AgentSummary       string
	FinalResponse      string
	TaskType           *string
	Status             string
	Tags               []string
}

type DreamingInput struct {
	UserID      string
	TriggerType string
	Scope       string
	Query       string
}

type DreamingResult struct {
	Session  store.DreamingSession   `json:"session"`
	Insights []store.DreamingInsight `json:"insights"`
}

func NewManager(profiles ProfileRetriever, semantic SemanticRetriever) *Manager {
	return &Manager{
		profiles: profiles,
		semantic: semantic,
	}
}

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

func (m *Manager) MatchSkills(ctx context.Context, userID, query, loadMode string, limit int) ([]store.SkillDisclosure, error) {
	if m.profiles == nil {
		return nil, nil
	}
	return m.profiles.MatchSkillDisclosures(ctx, userID, query, loadMode, limit)
}

func (m *Manager) EnqueueCuratorJob(ctx context.Context, userID, jobType string, payload []byte) (store.CuratorJob, error) {
	if m.profiles == nil {
		return store.CuratorJob{}, nil
	}
	return m.profiles.CreateCuratorJob(ctx, store.CuratorJobCreate{
		UserID:  userID,
		JobType: jobType,
		Payload: payload,
	})
}

func (m *Manager) RunDreaming(ctx context.Context, input DreamingInput) (DreamingResult, error) {
	if m.profiles == nil {
		return DreamingResult{}, nil
	}
	inputSummary := strings.TrimSpace(input.Query)
	if inputSummary == "" {
		inputSummary = "manual dreaming session"
	}
	session, err := m.profiles.CreateDreamingSession(ctx, store.DreamingSessionCreate{
		UserID:       input.UserID,
		TriggerType:  input.TriggerType,
		Scope:        input.Scope,
		InputSummary: &inputSummary,
	})
	if err != nil {
		return DreamingResult{}, err
	}
	insightContent := "建议复盘最近交互，识别可合并的偏好、可沉淀的技能和可归档的低价值记忆。"
	if strings.TrimSpace(input.Query) != "" {
		insightContent = "围绕「" + strings.TrimSpace(input.Query) + "」复盘最近记忆，建议检查是否存在可沉淀技能或可更新用户画像。"
	}
	insight, err := m.profiles.CreateDreamingInsight(ctx, store.DreamingInsightCreate{
		DreamingSessionID: session.ID,
		UserID:            input.UserID,
		InsightType:       "skill_candidate",
		SourceLayer:       "episodic",
		Content:           insightContent,
		TargetLayer:       stringPtr("procedural"),
		Confidence:        0.72,
	})
	if err != nil {
		_, _ = m.profiles.FinishDreamingSession(ctx, session.ID, input.UserID, "failed", stringPtr(err.Error()))
		return DreamingResult{}, err
	}
	outputSummary := "已生成 1 条 dreaming insight，等待后续 curator 或用户确认。"
	session, err = m.profiles.FinishDreamingSession(ctx, session.ID, input.UserID, "success", &outputSummary)
	if err != nil {
		return DreamingResult{}, err
	}
	return DreamingResult{Session: session, Insights: []store.DreamingInsight{insight}}, nil
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

type RoutePlan struct {
	IncludeWorking  bool
	IncludeProfile  bool
	IncludeSemantic bool
}

func Route(input RetrieveInput) RoutePlan {
	query := strings.ToLower(input.Query)
	return RoutePlan{
		IncludeWorking:  input.IncludeWorking,
		IncludeProfile:  input.IncludeProfile,
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
	if !input.IncludeWorking && !input.IncludeProfile && !input.IncludeSemantic {
		input.IncludeWorking = true
		input.IncludeProfile = true
		input.SemanticOnDemand = true
	}
	return input
}

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

func looksSemantic(query string) bool {
	keywords := []string{"文档", "资料", "知识库", "搜索", "检索", "rag", "论文", "课程", "根据"}
	for _, keyword := range keywords {
		if strings.Contains(query, keyword) {
			return true
		}
	}
	return false
}

func layerPriority(layer string) int {
	switch layer {
	case LayerWorking:
		return 0
	case LayerProfile:
		return 1
	case LayerProcedural:
		return 2
	case LayerEpisodic:
		return 3
	case LayerSemantic:
		return 4
	default:
		return 9
	}
}

func estimateTokens(content string) int {
	runes := len([]rune(content))
	if runes == 0 {
		return 0
	}
	return runes/4 + 1
}

func stringPtr(value string) *string {
	return &value
}

func sumTokens(chunks []Chunk) int {
	total := 0
	for _, chunk := range chunks {
		total += chunk.TokenCount
	}
	return total
}

func usedLayers(chunks []Chunk) []string {
	seen := map[string]bool{}
	var layers []string
	for _, chunk := range chunks {
		if !seen[chunk.Layer] {
			seen[chunk.Layer] = true
			layers = append(layers, chunk.Layer)
		}
	}
	return layers
}

func skippedLayers(plan RoutePlan, chunks []Chunk) []string {
	had := map[string]bool{}
	for _, chunk := range chunks {
		had[chunk.Layer] = true
	}
	var skipped []string
	if plan.IncludeWorking && !had[LayerWorking] {
		skipped = append(skipped, LayerWorking)
	}
	if plan.IncludeProfile && !had[LayerProfile] {
		skipped = append(skipped, LayerProfile)
	}
	if plan.IncludeSemantic && !had[LayerSemantic] {
		skipped = append(skipped, LayerSemantic)
	}
	return skipped
}
