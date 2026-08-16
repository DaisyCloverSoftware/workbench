package desktop

import (
	"strings"
	"sync/atomic"
)

// dashboardMCPRuntime is tri-state so pure model tests and headless callers can
// still use provider metadata until the production desktop has actually tried
// to bind its MCP listener: 0 unknown, 1 unavailable, 2 connected.
var dashboardMCPRuntime atomic.Int32

func setDashboardMCPRuntime(connected bool) {
	if connected {
		dashboardMCPRuntime.Store(2)
		return
	}
	dashboardMCPRuntime.Store(1)
}

func dashboardMCPReady(fallback bool) bool {
	switch dashboardMCPRuntime.Load() {
	case 1:
		return false
	case 2:
		return true
	default:
		return fallback
	}
}

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
