package api

import "github.com/gin-gonic/gin"

func registerToolRoutes(router *gin.RouterGroup, h Handlers) {
	router.GET("/tools", h.Tool.List)
	router.POST("/tools/:tool_name/call", h.Tool.Execute)
	router.GET("/conversations/:conversation_id/tool-calls", h.Tool.ConversationCalls)
	router.GET("/tool-calls/:tool_call_id", h.Tool.Call)
	router.POST("/tool-approval-requests/:approval_id/approve", h.Tool.Approve)
	router.POST("/tool-approval-requests/:approval_id/reject", h.Tool.Reject)
}
