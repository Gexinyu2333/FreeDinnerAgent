package app

import (
	"freedinner/backend/internal/store"

	"github.com/jackc/pgx/v5/pgxpool"
)

type stores struct {
	Users          *store.UserStore
	Sessions       *store.SessionStore
	AgentConfigs   *store.AgentConfigStore
	ModelProviders *store.ModelProviderStore
	Conversations  *store.ConversationStore
	LLMUsage       *store.LLMUsageStore
	Harness        *store.HarnessStore
	Contexts       *store.ContextStore
	Market         *store.MarketStore
	Knowledge      *store.KnowledgeStore
	Memory         *store.MemoryStore
	MCP            *store.MCPStore
	Tasks          *store.TaskStore
	ScheduledJobs  *store.ScheduledJobStore
	Tools          *store.ToolStore
	Channels       *store.ChannelStore
	Workspace      *store.WorkspaceStore
}

func newStores(db *pgxpool.Pool) stores {
	return stores{
		Users:          store.NewUserStore(db),
		Sessions:       store.NewSessionStore(db),
		AgentConfigs:   store.NewAgentConfigStore(db),
		ModelProviders: store.NewModelProviderStore(db),
		Conversations:  store.NewConversationStore(db),
		LLMUsage:       store.NewLLMUsageStore(db),
		Harness:        store.NewHarnessStore(db),
		Contexts:       store.NewContextStore(db),
		Market:         store.NewMarketStore(db),
		Knowledge:      store.NewKnowledgeStore(db),
		Memory:         store.NewMemoryStore(db),
		MCP:            store.NewMCPStore(db),
		Tasks:          store.NewTaskStore(db),
		ScheduledJobs:  store.NewScheduledJobStore(db),
		Tools:          store.NewToolStore(db),
		Channels:       store.NewChannelStore(db),
		Workspace:      store.NewWorkspaceStore(db),
	}
}
