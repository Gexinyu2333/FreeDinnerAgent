package memory

import (
	"context"
	"encoding/json"
	"testing"

	"freedinner/backend/internal/store"
)

type fakeProfiles struct {
	working []store.WorkingMemory
	profile []store.ProfileMemory
	skills  []store.SkillDisclosure
	logs    []store.MemoryRetrievalLogCreate
}

func (f *fakeProfiles) UpsertWorkingMemory(ctx context.Context, input store.WorkingMemoryUpsert) (store.WorkingMemory, error) {
	return store.WorkingMemory{MemoryKey: input.MemoryKey, MemoryValue: input.MemoryValue}, nil
}

func (f *fakeProfiles) ListWorkingMemories(ctx context.Context, userID, conversationID string, limit int) ([]store.WorkingMemory, error) {
	return f.working, nil
}

func (f *fakeProfiles) SearchProfileMemories(ctx context.Context, userID, query string, limit int) ([]store.ProfileMemory, error) {
	return f.profile, nil
}

func (f *fakeProfiles) CreateRetrievalLog(ctx context.Context, input store.MemoryRetrievalLogCreate) error {
	f.logs = append(f.logs, input)
	return nil
}

func (f *fakeProfiles) CreateEpisode(ctx context.Context, input store.EpisodeCreate) (store.Episode, error) {
	return store.Episode{ID: "episode-1", UserID: input.UserID, ConversationID: input.ConversationID, UserInput: input.UserInput}, nil
}

func (f *fakeProfiles) MatchSkillDisclosures(ctx context.Context, userID, query, loadMode string, limit int) ([]store.SkillDisclosure, error) {
	return f.skills, nil
}

func (f *fakeProfiles) CreateCuratorJob(ctx context.Context, input store.CuratorJobCreate) (store.CuratorJob, error) {
	return store.CuratorJob{ID: "job-1", UserID: input.UserID, JobType: input.JobType}, nil
}

func (f *fakeProfiles) CreateDreamingSession(ctx context.Context, input store.DreamingSessionCreate) (store.DreamingSession, error) {
	return store.DreamingSession{ID: "dream-1", UserID: input.UserID, TriggerType: input.TriggerType, Scope: input.Scope}, nil
}

func (f *fakeProfiles) FinishDreamingSession(ctx context.Context, sessionID, userID, status string, outputSummary *string) (store.DreamingSession, error) {
	return store.DreamingSession{ID: sessionID, UserID: userID, Status: status, OutputSummary: outputSummary}, nil
}

func (f *fakeProfiles) CreateDreamingInsight(ctx context.Context, input store.DreamingInsightCreate) (store.DreamingInsight, error) {
	return store.DreamingInsight{ID: "insight-1", DreamingSessionID: input.DreamingSessionID, UserID: input.UserID, InsightType: input.InsightType, Content: input.Content}, nil
}

type fakeSemantic struct {
	result SemanticSearchResult
	calls  int
}

func (f *fakeSemantic) SearchSemanticMemory(ctx context.Context, userID, query string, limit int) (SemanticSearchResult, error) {
	f.calls++
	return f.result, nil
}

func TestRetrieveBuildsUnifiedChunksAndLogs(t *testing.T) {
	docTitle := "课程资料"
	similarity := 0.88
	profiles := &fakeProfiles{
		working: []store.WorkingMemory{
			{ID: "wm-1", MemoryKey: "format", MemoryValue: "回答要简洁", Category: "constraint", TokenCount: 5},
		},
		profile: []store.ProfileMemory{
			{ID: "pm-1", MemoryType: "preference", Scope: "global", Title: "输出偏好", Content: "喜欢表格", Confidence: 0.9, Importance: 5, Metadata: json.RawMessage(`{}`)},
		},
	}
	semantic := &fakeSemantic{result: SemanticSearchResult{
		Mode: "vector",
		Chunks: []SemanticChunk{
			{ID: "kc-1", Visibility: "public", Content: "RAG 使用向量检索。", TokenCount: 8, Similarity: &similarity, DocumentTitle: &docTitle},
		},
	}}

	manager := NewManager(profiles, semantic)
	result, err := manager.Retrieve(context.Background(), RetrieveInput{
		UserID:          "user-1",
		ConversationID:  "conv-1",
		Query:           "根据知识库解释 RAG",
		MaxMemoryTokens: 100,
		LogRetrieval:    true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d: %#v", len(result.Chunks), result.Chunks)
	}
	if semantic.calls != 1 {
		t.Fatalf("expected semantic search on demand, got %d calls", semantic.calls)
	}
	if result.SemanticMode == nil || *result.SemanticMode != "vector" {
		t.Fatalf("expected vector semantic mode, got %#v", result.SemanticMode)
	}
	if len(profiles.logs) != len(result.Chunks) {
		t.Fatalf("expected retrieval logs for chunks, got %d", len(profiles.logs))
	}
}

func TestRouteOnlyUsesSemanticForKnowledgeLikeQueries(t *testing.T) {
	plan := Route(RetrieveInput{IncludeWorking: true, IncludeProfile: true, SemanticOnDemand: true, Query: "今天心情不错"})
	if plan.IncludeSemantic {
		t.Fatal("did not expect semantic retrieval for casual chat")
	}

	plan = Route(RetrieveInput{IncludeWorking: true, IncludeProfile: true, SemanticOnDemand: true, Query: "根据课程资料总结一下"})
	if !plan.IncludeSemantic {
		t.Fatal("expected semantic retrieval for document-like query")
	}
}

func TestCompressDeduplicatesAndPreservesHighPriority(t *testing.T) {
	chunks := []Chunk{
		{Layer: LayerSemantic, RefID: "s-1", Content: "semantic", Score: 0.95, TokenCount: 10},
		{Layer: LayerWorking, RefID: "w-1", Content: "working", Score: 0.95, TokenCount: 10},
		{Layer: LayerWorking, RefID: "w-1", Content: "duplicate", Score: 0.99, TokenCount: 10},
		{Layer: LayerProfile, RefID: "p-1", Content: "profile", Score: 0.50, TokenCount: 200},
	}

	result := Compress(chunks, 25)
	if len(result) != 2 {
		t.Fatalf("expected 2 chunks after token pruning, got %d: %#v", len(result), result)
	}
	if result[0].Layer != LayerWorking {
		t.Fatalf("expected working memory first on equal score, got %s", result[0].Layer)
	}
	if result[1].Layer != LayerSemantic {
		t.Fatalf("expected semantic second, got %s", result[1].Layer)
	}
}

func TestMatchSkillsDelegatesToProfileRetriever(t *testing.T) {
	profiles := &fakeProfiles{skills: []store.SkillDisclosure{{SkillID: "skill-1", Content: "先列步骤"}}}
	manager := NewManager(profiles, nil)
	skills, err := manager.MatchSkills(context.Background(), "user-1", "写周报", "light", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 1 || skills[0].SkillID != "skill-1" {
		t.Fatalf("unexpected skills: %#v", skills)
	}
}

func TestRunDreamingCreatesInsight(t *testing.T) {
	manager := NewManager(&fakeProfiles{}, nil)
	result, err := manager.RunDreaming(context.Background(), DreamingInput{UserID: "user-1", TriggerType: "manual", Scope: "user", Query: "周报流程"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Session.Status != "success" {
		t.Fatalf("expected success session, got %#v", result.Session)
	}
	if len(result.Insights) != 1 {
		t.Fatalf("expected one insight, got %d", len(result.Insights))
	}
}
