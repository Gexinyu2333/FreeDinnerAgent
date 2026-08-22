package api

import "github.com/gin-gonic/gin"

func registerWorkspaceRoutes(router *gin.RouterGroup, h Handlers) {
	router.GET("/me/workspace", h.Workspace.Status)
	router.POST("/me/workspace", h.Workspace.Enable)
	router.PATCH("/me/workspace", h.Workspace.UpdatePolicy)
	router.DELETE("/me/workspace", h.Workspace.Destroy)
	router.GET("/me/workspace/files", h.Workspace.ListFiles)
	router.GET("/me/workspace/files/content", h.Workspace.ReadFile)
	router.PUT("/me/workspace/files/content", h.Workspace.WriteFile)
	router.POST("/me/workspace/commands", h.Workspace.RunCommand)
	router.GET("/me/workspace/commands", h.Workspace.CommandRuns)
}
