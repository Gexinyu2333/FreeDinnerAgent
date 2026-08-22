package api

import "github.com/gin-gonic/gin"

func registerScheduledJobRoutes(router *gin.RouterGroup, h Handlers) {
	router.POST("/scheduled-agent-jobs", h.ScheduledJob.Create)
	router.GET("/scheduled-agent-jobs", h.ScheduledJob.List)
	router.GET("/scheduled-agent-job-templates", h.ScheduledJob.Templates)
	router.PATCH("/scheduled-agent-jobs/:job_id", h.ScheduledJob.Update)
	router.POST("/scheduled-agent-jobs/:job_id/pause", h.ScheduledJob.Pause)
	router.POST("/scheduled-agent-jobs/:job_id/resume", h.ScheduledJob.Resume)
	router.DELETE("/scheduled-agent-jobs/:job_id", h.ScheduledJob.Delete)
	router.GET("/scheduled-agent-jobs/:job_id/runs", h.ScheduledJob.Runs)
	router.POST("/scheduled-agent-jobs/:job_id/run-now", h.ScheduledJob.RunNow)
	router.GET("/scheduled-agent-job-runs/:run_id", h.ScheduledJob.Run)
}
