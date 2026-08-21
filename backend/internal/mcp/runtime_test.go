package mcp

import (
	"encoding/json"
	"testing"
)

func TestDiscoverConfiguredTools(t *testing.T) {
	def := ServerDefinition{
		Name:     "calendar",
		Metadata: json.RawMessage(`{"tools":[{"name":"list_events","description":"列出日程","permission_level":"readonly"}]}`),
	}
	setting := UserSetting{IsEnabled: true}
	tools := NewRuntime().DiscoverConfiguredTools(def, setting)
	if len(tools) != 1 {
		t.Fatalf("expected one tool, got %d", len(tools))
	}
	if tools[0].HandlerRef != "calendar.list_events" {
		t.Fatalf("unexpected handler ref: %s", tools[0].HandlerRef)
	}
}

func TestDiscoverConfiguredToolsRequiresEnabledSetting(t *testing.T) {
	def := ServerDefinition{Name: "calendar", Metadata: json.RawMessage(`{"tools":[{"name":"list_events"}]}`)}
	if got := NewRuntime().DiscoverConfiguredTools(def, UserSetting{}); len(got) != 0 {
		t.Fatalf("expected disabled setting to hide tools, got %#v", got)
	}
}
