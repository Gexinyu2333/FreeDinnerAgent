package api

import (
	"context"
	"net/http"

	"freedinner/backend/internal/auth"
	channelsvc "freedinner/backend/internal/channel"
	"freedinner/backend/internal/config"
	"freedinner/backend/internal/knowledge"
	"freedinner/backend/internal/llm"
	"freedinner/backend/internal/scheduler"
	"freedinner/backend/internal/secret"
	"freedinner/backend/internal/store"
	toolsvc "freedinner/backend/internal/tool"
	workspacesvc "freedinner/backend/internal/workspace"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Dependencies struct {
	Config config.Config
	DB     *pgxpool.Pool
}

func NewRouter(deps Dependencies) http.Handler {
	if deps.Config.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(RequestLogger())
	router.Use(CORS())

	router.GET("/healthz", healthHandler(deps))

	userStore := store.NewUserStore(deps.DB)
	sessionStore := store.NewSessionStore(deps.DB)
	authService := auth.NewService(userStore, sessionStore, deps.Config.JWTSecret)
	authHandler := NewAuthHandler(authService)
	agentConfigStore := store.NewAgentConfigStore(deps.DB)
	modelProviderStore := store.NewModelProviderStore(deps.DB)
	settingsHandler := NewSettingsHandler(agentConfigStore, modelProviderStore, secret.NewCrypto(deps.Config.APIKeyEncryptionKey))
	conversationStore := store.NewConversationStore(deps.DB)
	llmUsageStore := store.NewLLMUsageStore(deps.DB)
	harnessStore := store.NewHarnessStore(deps.DB)
	harnessHandler := NewHarnessHandler(harnessStore)
	knowledgeStore := store.NewKnowledgeStore(deps.DB)
	memoryStore := store.NewMemoryStore(deps.DB)
	memoryHandler := NewMemoryHandler(memoryStore)
	taskStore := store.NewTaskStore(deps.DB)
	scheduledJobStore := store.NewScheduledJobStore(deps.DB)
	toolStore := store.NewToolStore(deps.DB)
	channelStore := store.NewChannelStore(deps.DB)
	workspaceStore := store.NewWorkspaceStore(deps.DB)
	crypto := secret.NewCrypto(deps.Config.APIKeyEncryptionKey)
	llmService := llm.NewService(conversationStore, agentConfigStore, modelProviderStore, llmUsageStore, harnessStore, crypto, llm.NewOpenAIClient())
	conversationHandler := NewConversationHandler(conversationStore, llmService)
	knowledgeService := knowledge.NewService(knowledgeStore, agentConfigStore, modelProviderStore, crypto, llm.NewOpenAIClient())
	knowledgeHandler := NewKnowledgeHandler(knowledgeService)
	workspaceService := workspacesvc.NewService(workspaceStore, deps.Config.WorkspaceRoot, workspacesvc.RunnerOptions{
		DockerBinary: deps.Config.WorkspaceDockerBinary,
		PodmanBinary: deps.Config.WorkspacePodmanBinary,
		NsJailBinary: deps.Config.WorkspaceNsJailBinary,
		SandboxImage: deps.Config.WorkspaceSandboxImage,
	})
	workspaceHandler := NewWorkspaceHandler(workspaceService)
	toolService := toolsvc.NewService(toolStore, taskStore, memoryStore, knowledgeService, workspaceService)
	_ = toolService.EnsureBuiltins(context.Background())
	toolHandler := NewToolHandler(toolService)
	schedulerService := scheduler.NewService(scheduledJobStore, conversationStore)
	scheduledJobHandler := NewScheduledJobHandler(schedulerService)
	channelService := channelsvc.NewService(channelStore, conversationStore, crypto)
	_ = channelService.EnsureBuiltins(context.Background())
	channelHandler := NewChannelHandler(channelService)

	v1 := router.Group("/api/v1")
	v1.GET("/healthz", healthHandler(deps))
	v1.POST("/auth/register", authHandler.Register)
	v1.POST("/auth/login", authHandler.Login)
	v1.POST("/channels/:connection_id/webhook", channelHandler.Webhook)

	authenticated := v1.Group("")
	authenticated.Use(AuthMiddleware(deps.Config.JWTSecret))
	authenticated.GET("/me", authHandler.Me)
	authenticated.GET("/me/agent-config", settingsHandler.GetAgentConfig)
	authenticated.PATCH("/me/agent-config", settingsHandler.UpdateAgentConfig)
	authenticated.GET("/me/model-providers", settingsHandler.ListModelProviders)
	authenticated.POST("/me/model-providers", settingsHandler.CreateModelProvider)
	authenticated.PATCH("/me/model-providers/:provider_id", settingsHandler.UpdateModelProvider)
	authenticated.DELETE("/me/model-providers/:provider_id", settingsHandler.DeleteModelProvider)
	authenticated.POST("/conversations", conversationHandler.Create)
	authenticated.GET("/conversations", conversationHandler.List)
	authenticated.GET("/conversations/:conversation_id/messages", conversationHandler.Messages)
	authenticated.POST("/conversations/:conversation_id/messages", conversationHandler.SendMessage)
	authenticated.GET("/conversations/:conversation_id/agent-events", harnessHandler.Events)
	authenticated.GET("/conversations/:conversation_id/agent-turns/:turn_id", harnessHandler.Turn)
	authenticated.GET("/conversations/:conversation_id/agent-turns/:turn_id/loop-steps", harnessHandler.LoopSteps)
	authenticated.POST("/knowledge-documents", knowledgeHandler.Ingest)
	authenticated.GET("/knowledge-documents", knowledgeHandler.ListDocuments)
	authenticated.GET("/knowledge-search", knowledgeHandler.Search)
	authenticated.GET("/memory-types", memoryHandler.Types)
	authenticated.POST("/profile-memories", memoryHandler.CreateProfileMemory)
	authenticated.GET("/profile-memories", memoryHandler.ListProfileMemories)
	authenticated.GET("/profile-memory-search", memoryHandler.SearchProfileMemories)
	authenticated.GET("/tools", toolHandler.List)
	authenticated.POST("/tools/:tool_name/call", toolHandler.Execute)
	authenticated.POST("/scheduled-agent-jobs", scheduledJobHandler.Create)
	authenticated.GET("/scheduled-agent-jobs", scheduledJobHandler.List)
	authenticated.GET("/scheduled-agent-jobs/:job_id/runs", scheduledJobHandler.Runs)
	authenticated.POST("/scheduled-agent-jobs/:job_id/run-now", scheduledJobHandler.RunNow)
	authenticated.GET("/channel-providers", channelHandler.Providers)
	authenticated.POST("/me/channel-connections", channelHandler.CreateConnection)
	authenticated.GET("/me/channel-connections", channelHandler.Connections)
	authenticated.PATCH("/me/channel-connections/:connection_id/policies", channelHandler.UpsertPolicy)
	authenticated.GET("/me/channel-connections/:connection_id/external-conversations", channelHandler.ExternalConversations)
	authenticated.GET("/me/channel-connections/:connection_id/inbox-events", channelHandler.InboxEvents)
	authenticated.GET("/me/channel-connections/:connection_id/outbox-messages", channelHandler.OutboxMessages)
	authenticated.GET("/me/workspace", workspaceHandler.Status)
	authenticated.POST("/me/workspace", workspaceHandler.Enable)
	authenticated.GET("/me/workspace/files", workspaceHandler.ListFiles)
	authenticated.GET("/me/workspace/files/content", workspaceHandler.ReadFile)
	authenticated.PUT("/me/workspace/files/content", workspaceHandler.WriteFile)
	authenticated.POST("/me/workspace/commands", workspaceHandler.RunCommand)
	authenticated.GET("/me/workspace/commands", workspaceHandler.CommandRuns)

	return router
}
