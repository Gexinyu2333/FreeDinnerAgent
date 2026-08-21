package channel

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"freedinner/backend/internal/store"
)

func TestNormalizeOneBotPrivateMessage(t *testing.T) {
	raw := []byte(`{
		"post_type":"message",
		"message_type":"private",
		"message_id":1001,
		"user_id":12345,
		"raw_message":"你好",
		"sender":{"nickname":"小明"}
	}`)

	event, err := normalizeOneBot(raw, nil)
	if err != nil {
		t.Fatal(err)
	}
	if event.ExternalConversationType != "private_chat" || event.ExternalConversationID != "12345" {
		t.Fatalf("unexpected conversation: %#v", event)
	}
	if event.Text != "你好" || event.ExternalSenderName == nil || *event.ExternalSenderName != "小明" {
		t.Fatalf("unexpected normalized text/sender: %#v", event)
	}
}

func TestNormalizeOneBotGroupMessageStripsMention(t *testing.T) {
	botQQ := "999"
	raw := []byte(`{
		"post_type":"message",
		"message_type":"group",
		"message_id":"g-1",
		"group_id":67890,
		"user_id":12345,
		"raw_message":"[CQ:at,qq=999] 总结一下",
		"sender":{"card":"群昵称","nickname":"昵称"}
	}`)

	event, err := normalizeOneBot(raw, &botQQ)
	if err != nil {
		t.Fatal(err)
	}
	if event.ExternalConversationType != "group_chat" || event.ExternalConversationID != "67890" {
		t.Fatalf("unexpected group event: %#v", event)
	}
	if event.Text != "总结一下" {
		t.Fatalf("expected mention stripped, got %q", event.Text)
	}
}

func TestShouldTriggerPolicies(t *testing.T) {
	botQQ := "999"
	groupRaw := json.RawMessage(`{"raw_message":"[CQ:at,qq=999] hello"}`)
	event := normalizedEvent{ScopeType: "group_chat", Text: "hello", RawPayload: groupRaw}

	ok, reason := shouldTrigger(event, store.ChannelPolicy{Mode: "mention_only"}, &botQQ)
	if !ok || reason != "mention" {
		t.Fatalf("expected mention trigger, got ok=%v reason=%s", ok, reason)
	}

	ok, reason = shouldTrigger(normalizedEvent{ScopeType: "group_chat", Text: "普通聊天", RawPayload: json.RawMessage(`{}`)}, store.ChannelPolicy{Mode: "mention_only"}, &botQQ)
	if ok || reason != "mention_required" {
		t.Fatalf("expected mention required, got ok=%v reason=%s", ok, reason)
	}

	ok, reason = shouldTrigger(normalizedEvent{ScopeType: "private_chat", Text: "查一下论文"}, store.ChannelPolicy{Mode: "keyword", TriggerKeywords: []string{"论文"}}, nil)
	if !ok || reason != "keyword" {
		t.Fatalf("expected keyword trigger, got ok=%v reason=%s", ok, reason)
	}
}

func TestQuietHoursBlocksTrigger(t *testing.T) {
	quiet := json.RawMessage(`{"enabled":true,"timezone":"Asia/Shanghai","start":"23:00","end":"07:00"}`)
	now := time.Date(2026, 8, 21, 23, 30, 0, 0, time.FixedZone("Asia/Shanghai", 8*60*60))
	ok, reason := shouldTriggerAt(
		normalizedEvent{ScopeType: "private_chat", Text: "你好"},
		store.ChannelPolicy{Mode: "auto_reply", QuietHours: quiet},
		nil,
		now,
	)
	if ok || reason != "quiet_hours" {
		t.Fatalf("expected quiet hours block, got ok=%v reason=%s", ok, reason)
	}
}

func TestBuildOneBotSendPayload(t *testing.T) {
	payload, err := buildOneBotSendPayload(store.ExternalConversation{
		ExternalConversationID:   "67890",
		ExternalConversationType: "group_chat",
	}, "hello")
	if err != nil {
		t.Fatal(err)
	}
	text := string(payload)
	for _, expected := range []string{`"message_type":"group"`, `"group_id":"67890"`, `"message":"hello"`} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected %s in payload %s", expected, text)
		}
	}

	payload, err = buildOneBotSendPayload(store.ExternalConversation{
		ExternalConversationID:   "12345",
		ExternalConversationType: "private_chat",
	}, "hi")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(payload), `"message_type":"private"`) || !strings.Contains(string(payload), `"user_id":"12345"`) {
		t.Fatalf("unexpected private payload: %s", string(payload))
	}
}
