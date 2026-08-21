package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"freedinner/backend/internal/scheduler"
	"freedinner/backend/internal/store"

	"github.com/gin-gonic/gin"
)

type ScheduledJobHandler struct {
	scheduler *scheduler.Service
}

func NewScheduledJobHandler(schedulerService *scheduler.Service) *ScheduledJobHandler {
	return &ScheduledJobHandler{scheduler: schedulerService}
}

type createScheduledJobRequest struct {
	Title           string          `json:"title" binding:"required"`
	Description     *string         `json:"description"`
	JobType         string          `json:"job_type"`
	ScheduleKind    string          `json:"schedule_kind"`
	CronExpr        *string         `json:"cron_expr"`
	Timezone        string          `json:"timezone"`
	RunAtLocalTime  *string         `json:"run_at_local_time"`
	Weekdays        []int           `json:"weekdays"`
	PromptTemplate  string          `json:"prompt_template" binding:"required"`
	ContextPolicy   json.RawMessage `json:"context_policy"`
	ToolPolicy      json.RawMessage `json:"tool_policy"`
	DeliveryChannel string          `json:"delivery_channel"`
	Visibility      string          `json:"visibility"`
	Metadata        json.RawMessage `json:"metadata"`
}

func (h *ScheduledJobHandler) Create(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}

	var req createScheduledJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	title := strings.TrimSpace(req.Title)
	promptTemplate := strings.TrimSpace(req.PromptTemplate)
	if title == "" || promptTemplate == "" {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", "title and prompt_template are required")
		return
	}

	runAtLocalTime, err := normalizeLocalTime(req.RunAtLocalTime)
	if err != nil {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", "run_at_local_time must be HH:MM or HH:MM:SS")
		return
	}
	weekdays, err := normalizeWeekdays(req.Weekdays)
	if err != nil {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", "weekdays must contain integers from 1 to 7")
		return
	}

	timezone := strings.TrimSpace(req.Timezone)
	if timezone == "" {
		timezone = "Asia/Shanghai"
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", "timezone is invalid")
		return
	}

	job, err := h.scheduler.CreateJob(c.Request.Context(), scheduler.CreateJobInput{
		UserID:          userID,
		Title:           title,
		Description:     trimAPIString(req.Description),
		JobType:         scheduler.NormalizeJobType(req.JobType),
		ScheduleKind:    scheduler.NormalizeScheduleKind(req.ScheduleKind),
		CronExpr:        trimAPIString(req.CronExpr),
		Timezone:        timezone,
		RunAtLocalTime:  runAtLocalTime,
		Weekdays:        weekdays,
		PromptTemplate:  promptTemplate,
		ContextPolicy:   req.ContextPolicy,
		ToolPolicy:      req.ToolPolicy,
		DeliveryChannel: scheduler.NormalizeDeliveryChannel(req.DeliveryChannel),
		Visibility:      scheduler.NormalizeVisibility(req.Visibility),
		Metadata:        req.Metadata,
	})
	if err != nil {
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create scheduled agent job")
		return
	}

	OK(c, job)
}

func (h *ScheduledJobHandler) List(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}

	status := trimAPIString(queryStringPtr(c, "status"))
	limit := parseLimit(c.Query("limit"))
	jobs, err := h.scheduler.ListJobs(c.Request.Context(), userID, status, limit)
	if err != nil {
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list scheduled agent jobs")
		return
	}
	OK(c, jobs)
}

func (h *ScheduledJobHandler) Runs(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}

	runs, err := h.scheduler.ListRuns(c.Request.Context(), userID, c.Param("job_id"), parseLimit(c.Query("limit")))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			Error(c, http.StatusNotFound, "NOT_FOUND", "scheduled agent job not found")
			return
		}
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list scheduled agent job runs")
		return
	}
	OK(c, runs)
}

func (h *ScheduledJobHandler) RunNow(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}

	result, err := h.scheduler.RunNow(c.Request.Context(), userID, c.Param("job_id"))
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			Error(c, http.StatusNotFound, "NOT_FOUND", "scheduled agent job not found")
		case errors.Is(err, scheduler.ErrInactiveJob):
			Error(c, http.StatusConflict, "JOB_INACTIVE", "scheduled agent job is not active")
		default:
			Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to run scheduled agent job")
		}
		return
	}
	OK(c, result)
}

func normalizeLocalTime(value *string) (*string, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}
	raw := strings.TrimSpace(*value)
	for _, layout := range []string{"15:04:05", "15:04"} {
		parsed, err := time.Parse(layout, raw)
		if err == nil {
			normalized := parsed.Format("15:04:05")
			return &normalized, nil
		}
	}
	return nil, errors.New("invalid local time")
}

func normalizeWeekdays(values []int) ([]int32, error) {
	if len(values) == 0 {
		return []int32{}, nil
	}
	result := make([]int32, 0, len(values))
	seen := map[int]bool{}
	for _, value := range values {
		if value < 1 || value > 7 {
			return nil, errors.New("invalid weekday")
		}
		if seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, int32(value))
	}
	return result, nil
}

func trimAPIString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func queryStringPtr(c *gin.Context, key string) *string {
	value, ok := c.GetQuery(key)
	if !ok {
		return nil
	}
	return &value
}

func parseLimit(value string) int {
	limit, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 50
	}
	return limit
}
