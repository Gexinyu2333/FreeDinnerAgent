package channel

import (
	"encoding/json"
	"testing"
	"time"

	"freedinner/backend/internal/store"
)

func TestCombinedTrigger(t *testing.T) {
	bot := "999"
	for _, tc := range []struct {
		name, mode, text, raw, reason string
		want                          bool
	}{
		{"mention", "mention_or_keyword", "hello", `{"message":[{"type":"at","data":{"qq":"999"}}]}`, "mention", true},
		{"keyword", "mention_or_keyword", "查看天气", `{}`, "keyword", true},
		{"both", "mention_or_keyword", "天气", `{"raw_message":"[CQ:at,qq=999]天气"}`, "mention", true},
		{"neither", "mention_or_keyword", "hello", `{}`, "mention_or_keyword_missed", false},
		{"other mention", "mention_or_keyword", "hello", `{"message":[{"type":"at","data":{"qq":"123"}}]}`, "mention_or_keyword_missed", false},
		{"empty keyword does not match", "mention_or_keyword", "hello", `{}`, "mention_or_keyword_missed", false},
		{"mention only ignores keywords", "mention_only", "天气", `{}`, "mention_required", false},
		{"keyword only ignores mentions", "keyword", "hello", `{"raw_message":"[CQ:at,qq=999]"}`, "keyword_missed", false},
		{"disabled", "disabled", "天气", `{"raw_message":"[CQ:at,qq=999]"}`, "disabled", false},
		{"silent", "silent_listen", "天气", `{}`, "silent_listen", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			policy := store.ChannelPolicy{Mode: normalizeMode(tc.mode), TriggerKeywords: []string{"", "天气"}}
			event := normalizedEvent{ScopeType: "group_chat", Text: tc.text, RawPayload: json.RawMessage(tc.raw)}
			got, reason := shouldTrigger(event, policy, &bot)
			if got != tc.want || reason != tc.reason {
				t.Fatalf("got (%v, %q), want (%v, %q)", got, reason, tc.want, tc.reason)
			}
		})
	}
}

func TestCombinedTriggerRespectsQuietHours(t *testing.T) {
	policy := store.ChannelPolicy{
		Mode: "mention_or_keyword", TriggerKeywords: []string{"weather"},
		QuietHours: json.RawMessage(`{"enabled":true,"timezone":"UTC","start":"22:00","end":"08:00"}`),
	}
	event := normalizedEvent{ScopeType: "group_chat", Text: "weather"}
	now := time.Date(2026, 9, 5, 23, 0, 0, 0, time.UTC)
	if got, reason := shouldTriggerAt(event, policy, nil, now); got || reason != "quiet_hours" {
		t.Fatalf("got (%v, %q), want quiet_hours", got, reason)
	}
}
