package memory

import (
	"context"
	"encoding/json"
	"strings"

	"freedinner/backend/internal/store"
)

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

func (m *Manager) ApplyDreamingInsight(ctx context.Context, userID, insightID string) (ApplyDreamingInsightResult, error) {
	if m.profiles == nil {
		return ApplyDreamingInsightResult{}, nil
	}
	insight, err := m.profiles.FindDreamingInsight(ctx, userID, insightID)
	if err != nil {
		return ApplyDreamingInsightResult{}, err
	}
	result := ApplyDreamingInsightResult{Insight: insight}
	switch insight.InsightType {
	case "profile_update", "promote":
		if insight.TargetLayer != nil && *insight.TargetLayer == LayerProfile {
			memory, err := m.profiles.CreateProfileMemory(ctx, store.ProfileMemoryCreate{
				UserID:     userID,
				MemoryType: "other",
				Scope:      "global",
				Title:      "Dreaming insight",
				Content:    insight.Content,
				Confidence: insight.Confidence,
				Importance: 3,
			})
			if err != nil {
				return ApplyDreamingInsightResult{}, err
			}
			result.ProfileMemory = &memory
		}
	case "skill_candidate":
		if insight.TargetLayer != nil && *insight.TargetLayer == LayerProcedural && len(insight.SourceRefIDs) > 0 {
			outputTemplate := "复用该 insight 描述的流程，并在结尾给出下一步建议。"
			distilled, err := m.profiles.CreateSkillFromEpisode(ctx, store.SkillDistillationInput{
				UserID:         userID,
				EpisodeID:      insight.SourceRefIDs[0],
				Name:           "dreaming_skill_" + shortID(insight.ID),
				Description:    insight.Content,
				Keywords:       []string{"dreaming", "技能", "流程"},
				ReactSteps:     "1. 读取用户目标。\n2. 按 dreaming insight 中的流程处理。\n3. 输出结果和风险提醒。",
				OutputTemplate: &outputTemplate,
			})
			if err != nil {
				return ApplyDreamingInsightResult{}, err
			}
			result.Skill = &distilled.Skill
		}
	}
	payload, _ := json.Marshal(map[string]any{"dreaming_insight_id": insight.ID, "insight_type": insight.InsightType})
	job, _ := m.profiles.CreateCuratorJob(ctx, store.CuratorJobCreate{UserID: userID, JobType: "memory_consolidation", Payload: payload})
	if job.ID != "" {
		result.CuratorJob = &job
	}
	applied, err := m.profiles.SetDreamingInsightStatus(ctx, userID, insightID, "applied")
	if err != nil {
		return ApplyDreamingInsightResult{}, err
	}
	result.Insight = applied
	return result, nil
}

func (m *Manager) RejectDreamingInsight(ctx context.Context, userID, insightID string) (store.DreamingInsight, error) {
	if m.profiles == nil {
		return store.DreamingInsight{}, nil
	}
	return m.profiles.SetDreamingInsightStatus(ctx, userID, insightID, "rejected")
}
