package scheduler

import (
	"strings"

	"freedinner/backend/internal/store"
)

func scheduledUpdateFromCurrent(input UpdateJobInput, current store.ScheduledAgentJob) store.ScheduledAgentJobUpdate {
	update := store.ScheduledAgentJobUpdate{
		UserID:          input.UserID,
		JobID:           input.JobID,
		Title:           current.Title,
		Description:     current.Description,
		JobType:         current.JobType,
		ScheduleKind:    current.ScheduleKind,
		CronExpr:        current.CronExpr,
		Timezone:        current.Timezone,
		RunAtLocalTime:  current.RunAtLocalTime,
		Weekdays:        current.Weekdays,
		PromptTemplate:  current.PromptTemplate,
		ContextPolicy:   current.ContextPolicy,
		ToolPolicy:      current.ToolPolicy,
		DeliveryChannel: current.DeliveryChannel,
		Visibility:      current.Visibility,
		Metadata:        current.Metadata,
	}
	if input.Title != nil && strings.TrimSpace(*input.Title) != "" {
		update.Title = strings.TrimSpace(*input.Title)
	}
	if input.Description != nil {
		update.Description = trimOptional(*input.Description)
	}
	if input.JobType != nil {
		update.JobType = NormalizeJobType(*input.JobType)
	}
	if input.ScheduleKind != nil {
		update.ScheduleKind = NormalizeScheduleKind(*input.ScheduleKind)
	}
	if input.CronExpr != nil {
		update.CronExpr = trimOptional(*input.CronExpr)
	}
	if input.Timezone != nil && strings.TrimSpace(*input.Timezone) != "" {
		update.Timezone = strings.TrimSpace(*input.Timezone)
	}
	if input.RunAtLocalTime != nil {
		update.RunAtLocalTime = trimOptional(*input.RunAtLocalTime)
	}
	if input.WeekdaysSet {
		update.Weekdays = input.Weekdays
	}
	if input.PromptTemplate != nil && strings.TrimSpace(*input.PromptTemplate) != "" {
		update.PromptTemplate = strings.TrimSpace(*input.PromptTemplate)
	}
	if input.ContextPolicy != nil {
		update.ContextPolicy = *input.ContextPolicy
	}
	if input.ToolPolicy != nil {
		update.ToolPolicy = *input.ToolPolicy
	}
	if input.DeliveryChannel != nil {
		update.DeliveryChannel = NormalizeDeliveryChannel(*input.DeliveryChannel)
	}
	if input.Visibility != nil {
		update.Visibility = NormalizeVisibility(*input.Visibility)
	}
	if input.Metadata != nil {
		update.Metadata = *input.Metadata
	}
	return update
}
