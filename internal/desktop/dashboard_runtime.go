package desktop

import "strings"

func ApplyDashboardRuntimeConnections(snapshot DashboardSnapshot, mcpConnected bool) DashboardSnapshot {
	for i := range snapshot.Providers {
		provider := &snapshot.Providers[i]
		if provider.ID != "chatgpt" {
			continue
		}
		wasReady := provider.Ready
		provider.Ready = mcpConnected
		if mcpConnected {
			provider.Status = "MCP bridge connected"
		} else {
			provider.Status = "MCP bridge unavailable"
		}
		if provider.Ready && !wasReady {
			snapshot.ProviderReady++
		} else if !provider.Ready && wasReady && snapshot.ProviderReady > 0 {
			snapshot.ProviderReady--
		}
		provider.Status = strings.TrimSpace(provider.Status)
		break
	}
	return snapshot
}
