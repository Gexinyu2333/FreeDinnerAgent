package api

import "github.com/gin-gonic/gin"

func registerUserSettingsRoutes(router *gin.RouterGroup, h Handlers) {
	router.GET("/me", h.Auth.Me)
	router.GET("/me/agent-config", h.Settings.GetAgentConfig)
	router.PATCH("/me/agent-config", h.Settings.UpdateAgentConfig)
	router.GET("/me/model-providers", h.Settings.ListModelProviders)
	router.POST("/me/model-providers", h.Settings.CreateModelProvider)
	router.PATCH("/me/model-providers/:provider_id", h.Settings.UpdateModelProvider)
	router.DELETE("/me/model-providers/:provider_id", h.Settings.DeleteModelProvider)
}
