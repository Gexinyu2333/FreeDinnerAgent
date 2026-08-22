package api

import "github.com/gin-gonic/gin"

func registerKnowledgeMemoryRoutes(router *gin.RouterGroup, h Handlers) {
	router.POST("/knowledge-documents", h.Knowledge.Ingest)
	router.GET("/knowledge-documents", h.Knowledge.ListDocuments)
	router.GET("/knowledge-search", h.Knowledge.Search)
	router.GET("/memory-types", h.Memory.Types)
	router.POST("/profile-memories", h.Memory.CreateProfileMemory)
	router.GET("/profile-memories", h.Memory.ListProfileMemories)
	router.GET("/profile-memory-search", h.Memory.SearchProfileMemories)
	router.GET("/memory-context", h.Memory.Context)
	router.GET("/dreaming-insights", h.Memory.ListDreamingInsights)
	router.POST("/dreaming-insights/:insight_id/apply", h.Memory.ApplyDreamingInsight)
	router.POST("/dreaming-insights/:insight_id/reject", h.Memory.RejectDreamingInsight)
}
