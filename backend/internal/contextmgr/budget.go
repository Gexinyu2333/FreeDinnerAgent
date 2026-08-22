package contextmgr

type Budget struct {
	SystemTokens        int
	MemoryTokens        int
	SkillTokens         int
	ToolTokens          int
	SummaryTokens       int
	RecentMessageTokens int
}

func BudgetFor(maxTokens int) Budget {
	if maxTokens <= 0 {
		maxTokens = 12000
	}
	return Budget{
		SystemTokens:        percent(maxTokens, 10),
		MemoryTokens:        percent(maxTokens, 35),
		SkillTokens:         percent(maxTokens, 15),
		ToolTokens:          percent(maxTokens, 10),
		SummaryTokens:       percent(maxTokens, 10),
		RecentMessageTokens: percent(maxTokens, 15),
	}
}

func percent(value int, ratio int) int {
	return value * ratio / 100
}
