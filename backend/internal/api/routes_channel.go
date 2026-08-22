package api

import "github.com/gin-gonic/gin"

func registerChannelRoutes(router *gin.RouterGroup, h Handlers) {
	router.GET("/channel-providers", h.Channel.Providers)
	router.POST("/me/channel-connections", h.Channel.CreateConnection)
	router.GET("/me/channel-connections", h.Channel.Connections)
	router.PATCH("/me/channel-connections/:connection_id/policies", h.Channel.UpsertPolicy)
	router.GET("/me/channel-connections/:connection_id/external-conversations", h.Channel.ExternalConversations)
	router.GET("/me/channel-connections/:connection_id/inbox-events", h.Channel.InboxEvents)
	router.GET("/me/channel-connections/:connection_id/outbox-messages", h.Channel.OutboxMessages)
	router.POST("/channel-outbox-messages/:outbox_id/approve", h.Channel.ApproveOutbox)
	router.POST("/channel-outbox-messages/:outbox_id/cancel", h.Channel.CancelOutbox)
	router.POST("/channel-outbox-messages/:outbox_id/send", h.Channel.SendOutbox)
}
