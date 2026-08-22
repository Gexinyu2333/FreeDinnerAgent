package memory

import (
	"context"

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
	CreateProfileMemory(ctx context.Context, input store.ProfileMemoryCreate) (store.ProfileMemory, error)
	CreateRetrievalLog(ctx context.Context, input store.MemoryRetrievalLogCreate) error
	CreateEpisode(ctx context.Context, input store.EpisodeCreate) (store.Episode, error)
	SearchEpisodes(ctx context.Context, userID, query string, limit int) ([]store.EpisodeMatch, error)
	CreateSkillFromEpisode(ctx context.Context, input store.SkillDistillationInput) (store.SkillDistillationResult, error)
	MatchSkillDisclosures(ctx context.Context, userID, query, loadMode string, limit int) ([]store.SkillDisclosure, error)
	CreateCuratorJob(ctx context.Context, input store.CuratorJobCreate) (store.CuratorJob, error)
	CreateDreamingSession(ctx context.Context, input store.DreamingSessionCreate) (store.DreamingSession, error)
	FinishDreamingSession(ctx context.Context, sessionID, userID, status string, outputSummary *string) (store.DreamingSession, error)
	CreateDreamingInsight(ctx context.Context, input store.DreamingInsightCreate) (store.DreamingInsight, error)
	FindDreamingInsight(ctx context.Context, userID, insightID string) (store.DreamingInsight, error)
	SetDreamingInsightStatus(ctx context.Context, userID, insightID, status string) (store.DreamingInsight, error)
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
	EpisodicLimit    int
	IncludeWorking   bool
	IncludeProfile   bool
	IncludeEpisodic  bool
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

type SkillDistillationInput struct {
	UserID   string
	Episode  store.Episode
	Query    string
	Response string
	TaskType *string
}

type ApplyDreamingInsightResult struct {
	Insight       store.DreamingInsight `json:"insight"`
	ProfileMemory *store.ProfileMemory  `json:"profile_memory,omitempty"`
	Skill         *store.Skill          `json:"skill,omitempty"`
	CuratorJob    *store.CuratorJob     `json:"curator_job,omitempty"`
}

func NewManager(profiles ProfileRetriever, semantic SemanticRetriever) *Manager {
	return &Manager{
		profiles: profiles,
		semantic: semantic,
	}
}
