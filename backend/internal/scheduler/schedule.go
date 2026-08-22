package scheduler

import (
	"strconv"
	"strings"
	"time"
)

type ScheduleSpec struct {
	ScheduleKind   string
	CronExpr       *string
	Timezone       string
	RunAtLocalTime string
	Weekdays       []int32
	Now            time.Time
}

func ComputeNextRunAt(spec ScheduleSpec) *time.Time {
	kind := NormalizeScheduleKind(spec.ScheduleKind)
	if kind == "cron" {
		return nextCronRun(spec)
	}
	location, err := time.LoadLocation(defaultString(spec.Timezone, "Asia/Shanghai"))
	if err != nil {
		location = time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	now := spec.Now
	if now.IsZero() {
		now = time.Now()
	}
	localNow := now.In(location)
	hour, minute, second := parseRunAtLocalTime(spec.RunAtLocalTime)
	candidate := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), hour, minute, second, 0, location)

	switch kind {
	case "once":
		if candidate.After(localNow) {
			return timePtr(candidate)
		}
		return nil
	case "daily":
		if !candidate.After(localNow) {
			candidate = candidate.AddDate(0, 0, 1)
		}
		return timePtr(candidate)
	case "weekly":
		return nextWeeklyRun(localNow, candidate, spec.Weekdays)
	case "monthly":
		if !candidate.After(localNow) {
			candidate = candidate.AddDate(0, 1, 0)
		}
		return timePtr(candidate)
	default:
		return nil
	}
}

func nextCronRun(spec ScheduleSpec) *time.Time {
	if spec.CronExpr == nil || strings.TrimSpace(*spec.CronExpr) == "" {
		return nil
	}
	location, err := time.LoadLocation(defaultString(spec.Timezone, "Asia/Shanghai"))
	if err != nil {
		location = time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	now := spec.Now
	if now.IsZero() {
		now = time.Now()
	}
	localNow := now.In(location).Truncate(time.Minute)
	fields := strings.Fields(strings.TrimSpace(*spec.CronExpr))
	if len(fields) != 5 {
		return nil
	}
	minutes, ok := parseCronField(fields[0], 0, 59)
	if !ok {
		return nil
	}
	hours, ok := parseCronField(fields[1], 0, 23)
	if !ok {
		return nil
	}
	days, ok := parseCronField(fields[2], 1, 31)
	if !ok {
		return nil
	}
	months, ok := parseCronField(fields[3], 1, 12)
	if !ok {
		return nil
	}
	weekdays, ok := parseCronField(fields[4], 0, 7)
	if !ok {
		return nil
	}
	for offset := 1; offset <= 366*24*60; offset++ {
		candidate := localNow.Add(time.Duration(offset) * time.Minute)
		weekday := int(candidate.Weekday())
		if weekdays[7] && weekday == 0 {
			weekday = 7
		}
		if minutes[candidate.Minute()] && hours[candidate.Hour()] &&
			days[candidate.Day()] && months[int(candidate.Month())] &&
			(weekdays[int(candidate.Weekday())] || weekdays[weekday]) {
			return timePtr(candidate)
		}
	}
	return nil
}

func parseCronField(value string, min int, max int) (map[int]bool, bool) {
	result := map[int]bool{}
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, false
		}
		step := 1
		if strings.Contains(part, "/") {
			pieces := strings.Split(part, "/")
			if len(pieces) != 2 {
				return nil, false
			}
			part = pieces[0]
			parsedStep, err := strconv.Atoi(pieces[1])
			if err != nil || parsedStep <= 0 {
				return nil, false
			}
			step = parsedStep
		}
		start, end := min, max
		if part != "*" {
			if strings.Contains(part, "-") {
				pieces := strings.Split(part, "-")
				if len(pieces) != 2 {
					return nil, false
				}
				var err error
				start, err = strconv.Atoi(pieces[0])
				if err != nil {
					return nil, false
				}
				end, err = strconv.Atoi(pieces[1])
				if err != nil {
					return nil, false
				}
			} else {
				parsed, err := strconv.Atoi(part)
				if err != nil {
					return nil, false
				}
				start, end = parsed, parsed
			}
		}
		if start < min || end > max || start > end {
			return nil, false
		}
		for current := start; current <= end; current += step {
			result[current] = true
		}
	}
	return result, len(result) > 0
}

func nextWeeklyRun(localNow time.Time, todayAtTime time.Time, weekdays []int32) *time.Time {
	if len(weekdays) == 0 {
		weekdays = []int32{isoWeekday(localNow.Weekday())}
	}
	for dayOffset := 0; dayOffset <= 7; dayOffset++ {
		candidate := todayAtTime.AddDate(0, 0, dayOffset)
		if !containsWeekday(weekdays, isoWeekday(candidate.Weekday())) {
			continue
		}
		if candidate.After(localNow) {
			return timePtr(candidate)
		}
	}
	return nil
}

func parseRunAtLocalTime(value string) (int, int, int) {
	value = strings.TrimSpace(value)
	for _, layout := range []string{"15:04:05", "15:04"} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed.Hour(), parsed.Minute(), parsed.Second()
		}
	}
	return 9, 0, 0
}

func isoWeekday(value time.Weekday) int32 {
	if value == time.Sunday {
		return 7
	}
	return int32(value)
}

func containsWeekday(values []int32, target int32) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
