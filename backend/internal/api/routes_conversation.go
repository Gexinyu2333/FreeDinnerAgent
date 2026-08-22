package api

import "github.com/gin-gonic/gin"

func registerConversationRoutes(router *gin.RouterGroup, h Handlers) {
	router.POST("/conversations", h.Conversation.Create)
	router.GET("/conversations", h.Conversation.List)
	router.GET("/conversations/:conversation_id/messages", h.Conversation.Messages)
	router.POST("/conversations/:conversation_id/messages", h.Conversation.SendMessage)
	router.POST("/conversations/:conversation_id/compress", h.Context.ManualCompress)
	router.GET("/conversations/:conversation_id/agent-events", h.Harness.Events)
	router.GET("/conversations/:conversation_id/agent-turns/:turn_id", h.Harness.Turn)
	router.GET("/conversations/:conversation_id/agent-turns/:turn_id/loop-steps", h.Harness.LoopSteps)
}
