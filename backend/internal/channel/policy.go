package channel

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"freedinner/backend/internal/store"
)

func (s *Service) isRateLimited(ctx context.Context, connection store.ChannelConnection, event normalizedEvent, policy store.ChannelPolicy, now time.Time) (bool, error) {
	cfg := rateLimitConfig(policy)
	if cfg.CircuitBreaker.Enabled {
		return true, nil
	}
	for _, rule := range cfg.ScopeRules {
		if rule.MaxTriggers <= 0 || rule.Window <= 0 {
			continue
		}
		count, err := s.channels.CountRecentTriggeredInboxEvents(ctx, connection.UserID, connection.ID, event.ScopeType, event.ExternalScopeID, now.Add(-rule.Window))
		if err != nil {
			return false, err
		}
		if count >= rule.MaxTriggers {
			return true, nil
		}
	}
	for _, rule := range cfg.UserRules {
		if rule.MaxTriggers <= 0 || rule.Window <= 0 {
			continue
		}
		count, err := s.channels.CountRecentTriggeredInboxEventsForUser(ctx, connection.UserID, now.Add(-rule.Window))
		if err != nil {
			return false, err
		}
		if count >= rule.MaxTriggers {
			return true, nil
		}
	}
	return false, nil
}

type rateLimitRule struct {
	Window      time.Duration
	MaxTriggers int
}

type rateLimitConfigResult struct {
	ScopeRules     []rateLimitRule
	UserRules      []rateLimitRule
	CircuitBreaker struct {
		Enabled bool `json:"enabled"`
	} `json:"circuit_breaker"`
}

func rateLimitRules(policy store.ChannelPolicy) []rateLimitRule {
	return rateLimitConfig(policy).ScopeRules
}

func rateLimitConfig(policy store.ChannelPolicy) rateLimitConfigResult {
	cfg := rateLimitConfigResult{}
	if policy.RateLimitPerMinute > 0 {
		cfg.ScopeRules = append(cfg.ScopeRules, rateLimitRule{Window: time.Minute, MaxTriggers: policy.RateLimitPerMinute})
	}
	var metadata struct {
		RateLimits []struct {
			WindowSeconds int `json:"window_seconds"`
			MaxTriggers   int `json:"max_triggers"`
		} `json:"rate_limits"`
		UserRateLimits []struct {
			WindowSeconds int `json:"window_seconds"`
			MaxTriggers   int `json:"max_triggers"`
		} `json:"user_rate_limits"`
		CircuitBreaker struct {
			Enabled bool `json:"enabled"`
		} `json:"circuit_breaker"`
	}
	if len(policy.Metadata) > 0 {
		_ = json.Unmarshal(policy.Metadata, &metadata)
	}
	cfg.CircuitBreaker = metadata.CircuitBreaker
	for _, item := range metadata.RateLimits {
		if item.WindowSeconds <= 0 || item.MaxTriggers <= 0 {
			continue
		}
		cfg.ScopeRules = append(cfg.ScopeRules, rateLimitRule{
			Window:      time.Duration(item.WindowSeconds) * time.Second,
			MaxTriggers: item.MaxTriggers,
		})
	}
	for _, item := range metadata.UserRateLimits {
		if item.WindowSeconds <= 0 || item.MaxTriggers <= 0 {
			continue
		}
		cfg.UserRules = append(cfg.UserRules, rateLimitRule{
			Window:      time.Duration(item.WindowSeconds) * time.Second,
			MaxTriggers: item.MaxTriggers,
		})
	}
	return cfg
}

func rateLimitMetadata(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{}`)
	}
	var payload any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return json.RawMessage(`{}`)
	}
	return raw
}

func (s *Service) resolvePolicy(ctx context.Context, connection store.ChannelConnection, event normalizedEvent) store.ChannelPolicy {
	policy, err := s.channels.FindPolicy(ctx, connection.ID, event.ScopeType, event.ExternalScopeID)
	if err == nil {
		return policy
	}
	policy, err = s.channels.FindPolicy(ctx, connection.ID, event.ScopeType, nil)
	if err == nil {
		return policy
	}
	return store.ChannelPolicy{
		ScopeType:                  event.ScopeType,
		Mode:                       defaultMode(event.ScopeType),
		AllowMemoryWrite:           true,
		AllowToolUse:               true,
		RequireApprovalForOutbound: event.ScopeType == "group_chat",
		RateLimitPerMinute:         6,
	}
}

func shouldTrigger(event normalizedEvent, policy store.ChannelPolicy, botQQ *string) (bool, string) {
	return shouldTriggerAt(event, policy, botQQ, time.Now())
}

func shouldTriggerAt(event normalizedEvent, policy store.ChannelPolicy, botQQ *string, now time.Time) (bool, string) {
	if inQuietHours(policy.QuietHours, now) {
		return false, "quiet_hours"
	}
	switch policy.Mode {
	case "disabled", "silent_listen":
		return false, policy.Mode
	case "auto_reply":
		return true, "auto_reply"
	case "keyword":
		for _, keyword := range policy.TriggerKeywords {
			if keyword != "" && strings.Contains(event.Text, keyword) {
				return true, "keyword"
			}
		}
		return false, "keyword_missed"
	default:
		if event.ScopeType == "group_chat" {
			if botQQ != nil && strings.Contains(string(event.RawPayload), "[CQ:at,qq="+*botQQ+"]") {
				return true, "mention"
			}
			return false, "mention_required"
		}
		return true, "private_chat"
	}
}

type quietHoursConfig struct {
	Enabled  bool   `json:"enabled"`
	Timezone string `json:"timezone"`
	Start    string `json:"start"`
	End      string `json:"end"`
}

func inQuietHours(raw json.RawMessage, now time.Time) bool {
	if len(raw) == 0 || string(raw) == "{}" || string(raw) == "null" {
		return false
	}
	var cfg quietHoursConfig
	if err := json.Unmarshal(raw, &cfg); err != nil || !cfg.Enabled {
		return false
	}
	location := time.Local
	if strings.TrimSpace(cfg.Timezone) != "" {
		if loaded, err := time.LoadLocation(strings.TrimSpace(cfg.Timezone)); err == nil {
			location = loaded
		}
	}
	localNow := now.In(location)
	startMinute, ok := parseClockMinute(cfg.Start)
	if !ok {
		return false
	}
	endMinute, ok := parseClockMinute(cfg.End)
	if !ok {
		return false
	}
	currentMinute := localNow.Hour()*60 + localNow.Minute()
	if startMinute == endMinute {
		return true
	}
	if startMinute < endMinute {
		return currentMinute >= startMinute && currentMinute < endMinute
	}
	return currentMinute >= startMinute || currentMinute < endMinute
}

func parseClockMinute(value string) (int, bool) {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) < 2 {
		return 0, false
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return 0, false
	}
	minute, err := strconv.Atoi(parts[1])
	if err != nil || minute < 0 || minute > 59 {
		return 0, false
	}
	return hour*60 + minute, true
}

func defaultMode(scopeType string) string {
	if scopeType == "group_chat" {
		return "mention_only"
	}
	return "auto_reply"
}

func normalizeScopeType(value string) string {
	switch strings.TrimSpace(value) {
	case "group_chat", "all":
		return strings.TrimSpace(value)
	default:
		return "private_chat"
	}
}

func normalizeMode(value string) string {
	switch strings.TrimSpace(value) {
	case "disabled", "silent_listen", "mention_only", "keyword", "auto_reply":
		return strings.TrimSpace(value)
	default:
		return "mention_only"
	}
}
