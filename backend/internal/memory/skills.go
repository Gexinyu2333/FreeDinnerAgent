package memory

import (
	"context"
	"strings"

	"freedinner/backend/internal/store"
)

func (m *Manager) MatchSkills(ctx context.Context, userID, query, loadMode string, limit int) ([]store.SkillDisclosure, error) {
	if m.profiles == nil {
		return nil, nil
	}
	return m.profiles.MatchSkillDisclosures(ctx, userID, query, loadMode, limit)
}

func (m *Manager) DistillSkillFromEpisode(ctx context.Context, input SkillDistillationInput) (store.SkillDistillationResult, error) {
	if m.profiles == nil {
		return store.SkillDistillationResult{}, nil
	}
	name := skillName(input)
	description := "从一次成功交互中沉淀的可复用流程：" + trimText(input.Query, 80)
	outputTemplate := "按用户目标先给结论，再给步骤、结果和后续建议。"
	reactSteps := "1. 识别用户目标和约束。\n2. 检索相关记忆和历史经验。\n3. 判断是否需要工具；如需工具，先选择低风险工具。\n4. 汇总观察结果。\n5. 用清晰结构输出最终答复。"
	return m.profiles.CreateSkillFromEpisode(ctx, store.SkillDistillationInput{
		UserID:         input.UserID,
		EpisodeID:      input.Episode.ID,
		Name:           name,
		Description:    description,
		Keywords:       skillKeywords(input),
		ReactSteps:     reactSteps,
		OutputTemplate: &outputTemplate,
	})
}

func skillName(input SkillDistillationInput) string {
	base := "episode_skill"
	if input.TaskType != nil && strings.TrimSpace(*input.TaskType) != "" {
		base = strings.TrimSpace(*input.TaskType) + "_skill"
	}
	if input.Episode.ID != "" {
		id := input.Episode.ID
		if len(id) > 8 {
			id = id[:8]
		}
		base += "_" + id
	}
	return base
}

func skillKeywords(input SkillDistillationInput) []string {
	keywords := []string{"流程", "技能", "复用"}
	if input.TaskType != nil && strings.TrimSpace(*input.TaskType) != "" {
		keywords = append(keywords, strings.TrimSpace(*input.TaskType))
	}
	for _, term := range strings.Fields(input.Query) {
		term = strings.TrimSpace(term)
		if len([]rune(term)) >= 2 && len([]rune(term)) <= 12 {
			keywords = append(keywords, term)
		}
	}
	return keywords
}
