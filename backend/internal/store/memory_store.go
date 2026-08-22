package store

import (
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MemoryTypeDefinition struct {
	MemoryType      string    `json:"memory_type"`
	DisplayName     string    `json:"display_name"`
	Description     string    `json:"description"`
	ExtractionHint  string    `json:"extraction_hint"`
	RetrievalWeight float64   `json:"retrieval_weight"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type ProfileMemory struct {
	ID              string          `json:"id"`
	UserID          string          `json:"user_id"`
	MemoryType      string          `json:"memory_type"`
	Scope           string          `json:"scope"`
	Title           string          `json:"title"`
	Content         string          `json:"content"`
	Evidence        *string         `json:"evidence"`
	SourceMessageID *string         `json:"source_message_id"`
	Confidence      float64         `json:"confidence"`
	Importance      int             `json:"importance"`
	Status          string          `json:"status"`
	Metadata        json.RawMessage `json:"metadata"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type WorkingMemory struct {
	ID             string     `json:"id"`
	UserID         string     `json:"user_id"`
	ConversationID string     `json:"conversation_id"`
	MemoryKey      string     `json:"memory_key"`
	MemoryValue    string     `json:"memory_value"`
	Category       string     `json:"category"`
	TokenCount     int        `json:"token_count"`
	ExpiresAt      *time.Time `json:"expires_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type Episode struct {
	ID                 string          `json:"id"`
	UserID             string          `json:"user_id"`
	ConversationID     string          `json:"conversation_id"`
	UserMessageID      *string         `json:"user_message_id"`
	AssistantMessageID *string         `json:"assistant_message_id"`
	UserInput          string          `json:"user_input"`
	AgentSummary       string          `json:"agent_summary"`
	FinalResponse      string          `json:"final_response"`
	TaskType           *string         `json:"task_type"`
	Status             string          `json:"status"`
	Importance         int             `json:"importance"`
	TokenCount         int             `json:"token_count"`
	Metadata           json.RawMessage `json:"metadata"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

type EpisodeMatch struct {
	Episode
	Tags  []string `json:"tags"`
	Score float64  `json:"score"`
}

type SkillDisclosure struct {
	SkillID         string    `json:"skill_id"`
	SkillName       string    `json:"skill_name"`
	Description     string    `json:"description"`
	PermissionLevel string    `json:"permission_level"`
	VersionID       string    `json:"version_id"`
	Version         int       `json:"version"`
	DisclosureLevel string    `json:"disclosure_level"`
	Title           string    `json:"title"`
	Content         string    `json:"content"`
	TokenCount      int       `json:"token_count"`
	CreatedAt       time.Time `json:"created_at"`
}

type Skill struct {
	ID              string          `json:"id"`
	UserID          string          `json:"user_id"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	TriggerKeywords []string        `json:"trigger_keywords"`
	Scenario        *string         `json:"scenario"`
	Visibility      string          `json:"visibility"`
	PermissionLevel string          `json:"permission_level"`
	Status          string          `json:"status"`
	UseCount        int             `json:"use_count"`
	SuccessCount    int             `json:"success_count"`
	FailureCount    int             `json:"failure_count"`
	Metadata        json.RawMessage `json:"metadata"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type SkillVersion struct {
	ID                   string          `json:"id"`
	SkillID              string          `json:"skill_id"`
	Version              int             `json:"version"`
	ReactSteps           string          `json:"react_steps"`
	ToolSequence         json.RawMessage `json:"tool_sequence"`
	OutputTemplate       *string         `json:"output_template"`
	FallbackStrategy     *string         `json:"fallback_strategy"`
	CreatedFromEpisodeID *string         `json:"created_from_episode_id"`
	CreatedAt            time.Time       `json:"created_at"`
}

type SkillDistillationResult struct {
	Skill   Skill        `json:"skill"`
	Version SkillVersion `json:"version"`
}

type DreamingSession struct {
	ID            string     `json:"id"`
	UserID        string     `json:"user_id"`
	TriggerType   string     `json:"trigger_type"`
	Scope         string     `json:"scope"`
	Status        string     `json:"status"`
	InputSummary  *string    `json:"input_summary"`
	OutputSummary *string    `json:"output_summary"`
	StartedAt     *time.Time `json:"started_at"`
	FinishedAt    *time.Time `json:"finished_at"`
	CreatedAt     time.Time  `json:"created_at"`
}

type DreamingInsight struct {
	ID                string     `json:"id"`
	DreamingSessionID string     `json:"dreaming_session_id"`
	UserID            string     `json:"user_id"`
	InsightType       string     `json:"insight_type"`
	SourceLayer       string     `json:"source_layer"`
	SourceRefIDs      []string   `json:"source_ref_ids"`
	TargetLayer       *string    `json:"target_layer"`
	TargetRefID       *string    `json:"target_ref_id"`
	Content           string     `json:"content"`
	Confidence        float64    `json:"confidence"`
	Status            string     `json:"status"`
	CreatedAt         time.Time  `json:"created_at"`
	AppliedAt         *time.Time `json:"applied_at"`
}

type CuratorJob struct {
	ID           string          `json:"id"`
	UserID       string          `json:"user_id"`
	JobType      string          `json:"job_type"`
	Payload      json.RawMessage `json:"payload"`
	Status       string          `json:"status"`
	ErrorMessage *string         `json:"error_message"`
	CreatedAt    time.Time       `json:"created_at"`
	StartedAt    *time.Time      `json:"started_at"`
	FinishedAt   *time.Time      `json:"finished_at"`
}

type ProfileMemoryCreate struct {
	UserID          string
	MemoryType      string
	Scope           string
	Title           string
	Content         string
	Evidence        *string
	SourceMessageID *string
	Confidence      float64
	Importance      int
	Metadata        json.RawMessage
}

type EpisodeCreate struct {
	UserID             string
	ConversationID     string
	UserMessageID      *string
	AssistantMessageID *string
	UserInput          string
	AgentSummary       string
	FinalResponse      string
	TaskType           *string
	Status             string
	Importance         int
	TokenCount         int
	Metadata           json.RawMessage
	Tags               []string
}

type CuratorJobCreate struct {
	UserID  string
	JobType string
	Payload json.RawMessage
}

type DreamingSessionCreate struct {
	UserID       string
	TriggerType  string
	Scope        string
	InputSummary *string
}

type DreamingInsightCreate struct {
	DreamingSessionID string
	UserID            string
	InsightType       string
	SourceLayer       string
	SourceRefIDs      []string
	TargetLayer       *string
	TargetRefID       *string
	Content           string
	Confidence        float64
}

type SkillDistillationInput struct {
	UserID         string
	EpisodeID      string
	Name           string
	Description    string
	Keywords       []string
	ReactSteps     string
	OutputTemplate *string
}

type WorkingMemoryUpsert struct {
	UserID         string
	ConversationID string
	MemoryKey      string
	MemoryValue    string
	Category       string
	TokenCount     int
	ExpiresAt      *time.Time
}

type MemoryRetrievalLogCreate struct {
	UserID         string
	ConversationID string
	MessageID      *string
	MemoryLayer    string
	MemoryRefID    string
	Score          *float64
	TokenCount     int
	LoadMode       string
}

type MemoryStore struct {
	db *pgxpool.Pool
}

func NewMemoryStore(db *pgxpool.Pool) *MemoryStore {
	return &MemoryStore{db: db}
}
