package app

import (
	"context"
	"net/http"

	"freedinner/backend/internal/api"
	"freedinner/backend/internal/auth"
	channelsvc "freedinner/backend/internal/channel"
	"freedinner/backend/internal/config"
	"freedinner/backend/internal/contextmgr"
	"freedinner/backend/internal/knowledge"
	"freedinner/backend/internal/llm"
	marketsvc "freedinner/backend/internal/market"
	"freedinner/backend/internal/mcp"
	memorysvc "freedinner/backend/internal/memory"
	"freedinner/backend/internal/scheduler"
	"freedinner/backend/internal/secret"
	toolsvc "freedinner/backend/internal/tool"
	workspacesvc "freedinner/backend/internal/workspace"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Dependencies struct {
	Config  config.Config
	DB      *pgxpool.Pool
	Context context.Context
}

func NewHandler(deps Dependencies) http.Handler {
	appCtx := deps.Context
	if appCtx == nil {
		appCtx = context.Background()
	}

	stores := newStores(deps.DB)
	crypto := secret.NewCrypto(deps.Config.APIKeyEncryptionKey)
	openAIClient := llm.NewOpenAIClient()

	authService := auth.NewService(stores.Users, stores.Sessions, deps.Config.JWTSecret)
	contextBuilder := contextmgr.NewBuilder(stores.Contexts)
	contextCompressor := contextmgr.NewCompressor(stores.Contexts)
	marketService := marketsvc.NewService(stores.Market, stores.AgentConfigs)
	knowledgeService := knowledge.NewService(stores.Knowledge, stores.AgentConfigs, stores.ModelProviders, crypto, openAIClient)
	memoryManager := memorysvc.NewManager(stores.Memory, semanticMemoryAdapter{knowledge: knowledgeService})

	workspaceService := workspacesvc.NewService(stores.Workspace, deps.Config.WorkspaceRoot, workspacesvc.RunnerOptions{
		DockerBinary: deps.Config.WorkspaceDockerBinary,
		PodmanBinary: deps.Config.WorkspacePodmanBinary,
		NsJailBinary: deps.Config.WorkspaceNsJailBinary,
		SandboxImage: deps.Config.WorkspaceSandboxImage,
	})

	toolService := toolsvc.NewService(stores.Tools, stores.AgentConfigs, stores.Tasks, stores.Memory, knowledgeService, workspaceService)
	_ = toolService.EnsureBuiltins(context.Background())
	_, _ = mcp.NewRuntime().SyncConfiguredTools(context.Background(), stores.MCP, stores.Tools, 200)
	_ = syncToolMarketplaceItems(context.Background(), stores.Market, stores.Tools)

	llmService := llm.NewService(
		stores.Conversations,
		stores.AgentConfigs,
		stores.ModelProviders,
		stores.LLMUsage,
		stores.Harness,
		contextBuilder,
		contextCompressor,
		stores.Market,
		memoryManager,
		toolService,
		crypto,
		openAIClient,
	)

	schedulerService := scheduler.NewService(stores.ScheduledJobs, stores.Conversations, llmService)
	if deps.Config.SchedulerWorkerEnabled {
		schedulerService.StartWorker(appCtx, deps.Config.SchedulerPollInterval, 20)
	}

	channelService := channelsvc.NewService(stores.Channels, stores.Conversations, crypto, llmService)
	_ = channelService.EnsureBuiltins(context.Background())
	_ = syncChannelMarketplaceItems(context.Background(), stores.Market, stores.Channels)
	if deps.Config.ChannelSenderEnabled {
		channelService.StartSenderWorker(appCtx, deps.Config.ChannelSenderInterval, deps.Config.ChannelSenderBatchSize)
	}

	return api.NewRouter(api.RouterConfig{
		AppEnv:    deps.Config.AppEnv,
		JWTSecret: deps.Config.JWTSecret,
		DB:        deps.DB,
		Handlers: api.Handlers{
			Auth:         api.NewAuthHandler(authService),
			Settings:     api.NewSettingsHandler(stores.AgentConfigs, stores.ModelProviders, crypto),
			Conversation: api.NewConversationHandler(stores.Conversations, llmService),
			Context:      api.NewContextHandler(stores.Conversations, contextCompressor),
			Harness:      api.NewHarnessHandler(stores.Harness),
			Knowledge:    api.NewKnowledgeHandler(knowledgeService),
			Memory:       api.NewMemoryHandler(stores.Memory, memoryManager),
			Market:       api.NewMarketHandler(marketService),
			Tool:         api.NewToolHandler(toolService),
			ScheduledJob: api.NewScheduledJobHandler(schedulerService),
			Channel:      api.NewChannelHandler(channelService),
			Workspace:    api.NewWorkspaceHandler(workspaceService),
		},
	})
}
