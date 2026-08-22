package app

import "testing"

func TestNewStoresInitializesEveryStore(t *testing.T) {
	stores := newStores(nil)

	if stores.Users == nil ||
		stores.Sessions == nil ||
		stores.AgentConfigs == nil ||
		stores.ModelProviders == nil ||
		stores.Conversations == nil ||
		stores.LLMUsage == nil ||
		stores.Harness == nil ||
		stores.Contexts == nil ||
		stores.Market == nil ||
		stores.Knowledge == nil ||
		stores.Memory == nil ||
		stores.MCP == nil ||
		stores.Tasks == nil ||
		stores.ScheduledJobs == nil ||
		stores.Tools == nil ||
		stores.Channels == nil ||
		stores.Workspace == nil {
		t.Fatalf("newStores left a store nil: %#v", stores)
	}
}
