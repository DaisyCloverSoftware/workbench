package desktop

import "testing"

func TestApplyDashboardRuntimeConnectionsMarksChatGPTReady(t *testing.T) {
	snapshot := DashboardSnapshot{
		ProviderReady: 1,
		ProviderTotal: 2,
		Providers: []DashboardProviderItem{
			{ID: "chatgpt", Name: "ChatGPT Chat", Ready: false, Status: "MCP bridge ready"},
			{ID: "claude", Name: "Claude", Ready: true, Status: "connected"},
		},
	}
	updated := ApplyDashboardRuntimeConnections(snapshot, true)
	if updated.ProviderReady != 2 {
		t.Fatalf("expected 2 ready providers, got %d", updated.ProviderReady)
	}
	if !updated.Providers[0].Ready || updated.Providers[0].Status != "MCP bridge connected" {
		t.Fatalf("unexpected ChatGPT state: %+v", updated.Providers[0])
	}
}

func TestApplyDashboardRuntimeConnectionsMarksChatGPTOffline(t *testing.T) {
	snapshot := DashboardSnapshot{
		ProviderReady: 2,
		Providers: []DashboardProviderItem{{ID: "chatgpt", Ready: true}},
	}
	updated := ApplyDashboardRuntimeConnections(snapshot, false)
	if updated.ProviderReady != 1 || updated.Providers[0].Ready {
		t.Fatalf("unexpected offline state: %+v", updated)
	}
}
