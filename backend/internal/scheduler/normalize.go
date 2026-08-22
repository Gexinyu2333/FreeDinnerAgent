package scheduler

import (
	"strings"
	"time"
)

func NormalizeJobType(value string) string {
	switch strings.TrimSpace(value) {
	case "daily_brief", "weekly_review", "follow_up_monitor", "reminder", "content_digest", "social_assist", "custom":
		return strings.TrimSpace(value)
	default:
		return "custom"
	}
}

func NormalizeScheduleKind(value string) string {
	switch strings.TrimSpace(value) {
	case "once", "daily", "weekly", "monthly", "cron":
		return strings.TrimSpace(value)
	default:
		return "once"
	}
}

func NormalizeDeliveryChannel(value string) string {
	switch strings.TrimSpace(value) {
	case "email", "webhook":
		return strings.TrimSpace(value)
	default:
		return "in_app"
	}
}

func NormalizeVisibility(value string) string {
	if strings.TrimSpace(value) == "public_template" {
		return "public_template"
	}
	return "private"
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func trimOptional(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func timePtr(value time.Time) *time.Time {
	return &value
}
