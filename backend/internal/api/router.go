package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RouterConfig struct {
	AppEnv    string
	JWTSecret string
	DB        *pgxpool.Pool
	Handlers  Handlers
}

type Handlers struct {
	Auth         *AuthHandler
	Settings     *SettingsHandler
	Conversation *ConversationHandler
	Context      *ContextHandler
	Harness      *HarnessHandler
	Knowledge    *KnowledgeHandler
	Memory       *MemoryHandler
	Market       *MarketHandler
	Tool         *ToolHandler
	ScheduledJob *ScheduledJobHandler
	Channel      *ChannelHandler
	Workspace    *WorkspaceHandler
}

func NewRouter(cfg RouterConfig) http.Handler {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(RequestLogger())
	router.Use(CORS())

	router.GET("/healthz", healthHandler(cfg.AppEnv, cfg.DB))

	v1 := router.Group("/api/v1")
	v1.GET("/healthz", healthHandler(cfg.AppEnv, cfg.DB))
	registerPublicRoutes(v1, cfg.Handlers)

	authenticated := v1.Group("")
	authenticated.Use(AuthMiddleware(cfg.JWTSecret))
	registerAuthenticatedRoutes(authenticated, cfg.Handlers)

	return router
}

func registerPublicRoutes(router *gin.RouterGroup, h Handlers) {
	router.POST("/auth/register", h.Auth.Register)
	router.POST("/auth/login", h.Auth.Login)
	router.POST("/channels/:connection_id/webhook", h.Channel.Webhook)
}

func registerAuthenticatedRoutes(router *gin.RouterGroup, h Handlers) {
	registerUserSettingsRoutes(router, h)
	registerConversationRoutes(router, h)
	registerKnowledgeMemoryRoutes(router, h)
	registerMarketRoutes(router, h)
	registerToolRoutes(router, h)
	registerScheduledJobRoutes(router, h)
	registerChannelRoutes(router, h)
	registerWorkspaceRoutes(router, h)
}
