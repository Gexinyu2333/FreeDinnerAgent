package api

import "github.com/gin-gonic/gin"

func registerMarketRoutes(router *gin.RouterGroup, h Handlers) {
	router.GET("/marketplace-items", h.Market.ListItems)
	router.POST("/marketplace-items/:item_id/install", h.Market.Install)
	router.POST("/marketplace-items/:item_id/rate", h.Market.Rate)
	router.POST("/capability-installs/:install_id/enable", h.Market.EnableInstall)
	router.POST("/capability-installs/:install_id/disable", h.Market.DisableInstall)
	router.POST("/agent-capability-bindings", h.Market.Bind)
	router.POST("/agent-capability-bindings/:binding_id/enable", h.Market.EnableBinding)
	router.POST("/agent-capability-bindings/:binding_id/disable", h.Market.DisableBinding)
	router.POST("/system-prompt-templates", h.Market.CreatePromptTemplate)
	router.POST("/system-prompt-templates/preview", h.Market.PreviewPromptTemplate)
	router.POST("/system-prompt-template-versions/:version_id/fork", h.Market.ForkPromptTemplate)
}
