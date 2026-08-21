package scheduler

import (
	"strings"
	"testing"
	"time"

	"freedinner/backend/internal/store"
)

func TestDefaultTemplates(t *testing.T) {
	templates := DefaultTemplates()
	if len(templates) != 3 {
		t.Fatalf("expected 3 templates, got %d", len(templates))
	}
	if templates[0].ID != "daily_brief" {
		t.Fatalf("expected daily_brief first, got %q", templates[0].ID)
	}
	if templates[0].ScheduleKind != "weekly" || len(templates[0].Weekdays) != 5 {
		t.Fatalf("daily brief should run on workdays: %#v", templates[0])
	}
}

func TestComputeNextRunAtDaily(t *testing.T) {
	location := time.FixedZone("Asia/Shanghai", 8*60*60)
	now := time.Date(2026, 8, 21, 7, 30, 0, 0, location)
	next := ComputeNextRunAt(ScheduleSpec{
		ScheduleKind:   "daily",
		Timezone:       "Asia/Shanghai",
		RunAtLocalTime: "08:00:00",
		Now:            now,
	})
	if next == nil {
		t.Fatal("expected next run")
	}
	expected := time.Date(2026, 8, 21, 8, 0, 0, 0, location)
	if !next.Equal(expected) {
		t.Fatalf("expected %s, got %s", expected, next)
	}

	next = ComputeNextRunAt(ScheduleSpec{
		ScheduleKind:   "daily",
		Timezone:       "Asia/Shanghai",
		RunAtLocalTime: "08:00:00",
		Now:            time.Date(2026, 8, 21, 9, 0, 0, 0, location),
	})
	expected = time.Date(2026, 8, 22, 8, 0, 0, 0, location)
	if next == nil || !next.Equal(expected) {
		t.Fatalf("expected %s, got %v", expected, next)
	}
}

func TestComputeNextRunAtWeekly(t *testing.T) {
	location := time.FixedZone("Asia/Shanghai", 8*60*60)
	now := time.Date(2026, 8, 21, 15, 0, 0, 0, location) // Friday
	next := ComputeNextRunAt(ScheduleSpec{
		ScheduleKind:   "weekly",
		Timezone:       "Asia/Shanghai",
		RunAtLocalTime: "16:00:00",
		Weekdays:       []int32{5},
		Now:            now,
	})
	expected := time.Date(2026, 8, 21, 16, 0, 0, 0, location)
	if next == nil || !next.Equal(expected) {
		t.Fatalf("expected %s, got %v", expected, next)
	}

	next = ComputeNextRunAt(ScheduleSpec{
		ScheduleKind:   "weekly",
		Timezone:       "Asia/Shanghai",
		RunAtLocalTime: "16:00:00",
		Weekdays:       []int32{5},
		Now:            time.Date(2026, 8, 21, 17, 0, 0, 0, location),
	})
	expected = time.Date(2026, 8, 28, 16, 0, 0, 0, location)
	if next == nil || !next.Equal(expected) {
		t.Fatalf("expected %s, got %v", expected, next)
	}
}

func TestComputeNextRunAtMonthly(t *testing.T) {
	location := time.FixedZone("Asia/Shanghai", 8*60*60)
	next := ComputeNextRunAt(ScheduleSpec{
		ScheduleKind:   "monthly",
		Timezone:       "Asia/Shanghai",
		RunAtLocalTime: "18:30:00",
		Now:            time.Date(2026, 8, 21, 18, 0, 0, 0, location),
	})
	expected := time.Date(2026, 8, 21, 18, 30, 0, 0, location)
	if next == nil || !next.Equal(expected) {
		t.Fatalf("expected %s, got %v", expected, next)
	}

	next = ComputeNextRunAt(ScheduleSpec{
		ScheduleKind:   "monthly",
		Timezone:       "Asia/Shanghai",
		RunAtLocalTime: "18:30:00",
		Now:            time.Date(2026, 8, 21, 19, 0, 0, 0, location),
	})
	expected = time.Date(2026, 9, 21, 18, 30, 0, 0, location)
	if next == nil || !next.Equal(expected) {
		t.Fatalf("expected %s, got %v", expected, next)
	}
}

func TestComputeNextRunAtOnceReturnsNilAfterMissedTime(t *testing.T) {
	location := time.FixedZone("Asia/Shanghai", 8*60*60)
	next := ComputeNextRunAt(ScheduleSpec{
		ScheduleKind:   "once",
		Timezone:       "Asia/Shanghai",
		RunAtLocalTime: "08:00:00",
		Now:            time.Date(2026, 8, 21, 9, 0, 0, 0, location),
	})
	if next != nil {
		t.Fatalf("expected nil after missed once schedule, got %s", next)
	}
}

func TestComputeNextRunAtCronWithoutExpression(t *testing.T) {
	next := ComputeNextRunAt(ScheduleSpec{ScheduleKind: "cron", Now: time.Now()})
	if next != nil {
		t.Fatalf("expected nil for cron without expression, got %s", next)
	}
}

func TestScheduledRunSummary(t *testing.T) {
	if got := scheduledRunSummary("每日简报", "manual_run"); !strings.HasPrefix(got, "已手动触发心跳任务") {
		t.Fatalf("unexpected manual summary: %q", got)
	}
	if got := scheduledRunSummary("每日简报", "schedule_due"); !strings.HasPrefix(got, "心跳任务") {
		t.Fatalf("unexpected scheduled summary: %q", got)
	}
}

func TestComputeNextRunAtCron(t *testing.T) {
	expr := "*/15 9-10 * * 1-5"
	now := time.Date(2026, 8, 21, 9, 7, 0, 0, time.FixedZone("Asia/Shanghai", 8*60*60))
	next := ComputeNextRunAt(ScheduleSpec{
		ScheduleKind: "cron",
		CronExpr:     &expr,
		Timezone:     "Asia/Shanghai",
		Now:          now,
	})
	if next == nil {
		t.Fatal("expected next cron run")
	}
	if next.Minute() != 15 || next.Hour() != 9 {
		t.Fatalf("expected 09:15, got %s", next.Format(time.RFC3339))
	}
}

func TestFailureStatus(t *testing.T) {
	if got := failureStatus(store.ScheduledAgentJob{Status: "active"}); got != "failed" {
		t.Fatalf("expected failed, got %q", got)
	}
	if got := failureStatus(store.ScheduledAgentJob{Status: "paused"}); got != "paused_after_failures" {
		t.Fatalf("expected paused_after_failures, got %q", got)
	}
}

func TestScheduledUpdateFromCurrentPreservesOmittedFields(t *testing.T) {
	description := "old description"
	runAt := "08:00:00"
	current := store.ScheduledAgentJob{
		ID:              "job-1",
		UserID:          "user-1",
		Title:           "每日简报",
		Description:     &description,
		JobType:         "daily_brief",
		ScheduleKind:    "weekly",
		Timezone:        "Asia/Shanghai",
		RunAtLocalTime:  &runAt,
		Weekdays:        []int32{1, 2, 3, 4, 5},
		PromptTemplate:  "old prompt",
		DeliveryChannel: "in_app",
		Visibility:      "private",
	}
	title := "新的每日简报"
	update := scheduledUpdateFromCurrent(UpdateJobInput{
		UserID: "user-1",
		JobID:  "job-1",
		Title:  &title,
	}, current)

	if update.Title != "新的每日简报" {
		t.Fatalf("expected title update, got %q", update.Title)
	}
	if update.JobType != "daily_brief" || update.ScheduleKind != "weekly" {
		t.Fatalf("expected omitted schedule fields to be preserved: %#v", update)
	}
	if update.RunAtLocalTime == nil || *update.RunAtLocalTime != "08:00:00" {
		t.Fatalf("expected run time to be preserved: %#v", update.RunAtLocalTime)
	}
}

func TestNormalizeHelpers(t *testing.T) {
	if got := NormalizeJobType("daily_brief"); got != "daily_brief" {
		t.Fatalf("expected daily_brief, got %q", got)
	}
	if got := NormalizeJobType("unknown"); got != "custom" {
		t.Fatalf("expected custom fallback, got %q", got)
	}
	if hour, minute, second := parseRunAtLocalTime("07:05"); hour != 7 || minute != 5 || second != 0 {
		t.Fatalf("unexpected parsed time: %02d:%02d:%02d", hour, minute, second)
	}
	if got := isoWeekday(time.Sunday); got != 7 {
		t.Fatalf("expected Sunday iso weekday 7, got %d", got)
	}
}
