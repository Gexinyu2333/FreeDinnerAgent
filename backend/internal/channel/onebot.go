package channel

import (
	"encoding/json"
	"strconv"
	"strings"

	"freedinner/backend/internal/store"
)

type oneBotEvent struct {
	PostType    string          `json:"post_type"`
	MessageType string          `json:"message_type"`
	MessageID   any             `json:"message_id"`
	UserID      any             `json:"user_id"`
	GroupID     any             `json:"group_id"`
	RawMessage  string          `json:"raw_message"`
	Message     json.RawMessage `json:"message"`
	Sender      struct {
		Nickname string `json:"nickname"`
		Card     string `json:"card"`
	} `json:"sender"`
}

func normalizeOneBot(rawPayload []byte, botQQ *string) (normalizedEvent, error) {
	var raw oneBotEvent
	if err := json.Unmarshal(rawPayload, &raw); err != nil {
		return normalizedEvent{}, err
	}
	payload := json.RawMessage(rawPayload)
	if raw.PostType != "message" {
		eventID := eventID(raw.MessageID)
		return normalizedEvent{EventType: "system", ExternalEventID: eventID, Text: raw.PostType, RawPayload: payload}, nil
	}

	text := strings.TrimSpace(raw.RawMessage)
	if text == "" {
		text = strings.TrimSpace(string(raw.Message))
	}
	text = summarizeOneBotMessage(text)
	senderID := valueToString(raw.UserID)
	senderName := strings.TrimSpace(raw.Sender.Card)
	if senderName == "" {
		senderName = strings.TrimSpace(raw.Sender.Nickname)
	}
	eventID := eventID(raw.MessageID)
	senderNamePtr := stringPtrOrNil(senderName)
	senderIDPtr := stringPtrOrNil(senderID)

	if raw.MessageType == "group" {
		groupID := valueToString(raw.GroupID)
		title := "群聊 " + groupID
		return normalizedEvent{
			EventType:                "message_created",
			ExternalEventID:          eventID,
			ExternalConversationID:   groupID,
			ExternalConversationType: "group_chat",
			ExternalTitle:            &title,
			ExternalSenderID:         senderIDPtr,
			ExternalSenderName:       senderNamePtr,
			Text:                     stripBotMention(text, botQQ),
			ScopeType:                "group_chat",
			ExternalScopeID:          &groupID,
			RawPayload:               payload,
		}, nil
	}

	title := "私聊 " + senderID
	return normalizedEvent{
		EventType:                "message_created",
		ExternalEventID:          eventID,
		ExternalConversationID:   senderID,
		ExternalConversationType: "private_chat",
		ExternalTitle:            &title,
		ExternalSenderID:         senderIDPtr,
		ExternalSenderName:       senderNamePtr,
		Text:                     text,
		ScopeType:                "private_chat",
		ExternalScopeID:          &senderID,
		RawPayload:               payload,
	}, nil
}

func summarizeOneBotMessage(text string) string {
	replacements := map[string]string{
		"[CQ:image":  "[图片附件]",
		"[CQ:file":   "[文件附件]",
		"[CQ:record": "[语音附件]",
		"[CQ:video":  "[视频附件]",
	}
	summary := text
	for marker, label := range replacements {
		for {
			start := strings.Index(summary, marker)
			if start < 0 {
				break
			}
			end := strings.Index(summary[start:], "]")
			if end < 0 {
				summary = summary[:start] + label
				break
			}
			summary = summary[:start] + label + summary[start+end+1:]
		}
	}
	return strings.TrimSpace(summary)
}

func buildOneBotSendPayload(external store.ExternalConversation, content string) (json.RawMessage, error) {
	payload := map[string]any{
		"message": content,
	}
	switch external.ExternalConversationType {
	case "group_chat":
		payload["message_type"] = "group"
		payload["group_id"] = external.ExternalConversationID
	default:
		payload["message_type"] = "private"
		payload["user_id"] = external.ExternalConversationID
	}
	return json.Marshal(payload)
}

func eventID(value any) *string {
	stringValue := valueToString(value)
	if stringValue == "" {
		return nil
	}
	return &stringValue
}

func valueToString(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case int:
		return strconv.Itoa(v)
	default:
		return ""
	}
}

func stringPtrOrNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func stripBotMention(text string, botQQ *string) string {
	if botQQ == nil || *botQQ == "" {
		return strings.TrimSpace(text)
	}
	return strings.TrimSpace(strings.ReplaceAll(text, "[CQ:at,qq="+*botQQ+"]", ""))
}
