//go:build windows

package desktop

import (
	"testing"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestApplyRunnerProviderDashboardUsesActualClusterWorkers(t *testing.T) {
	snapshot := DashboardSnapshot{
		ProviderReady: 1,
		ProviderTotal: 3,
		Providers: []DashboardProviderItem{
			{ID: "workbench-runner", Name: "Workbench Cluster Runner", Ready: true, Cost: core.CostIncluded},
			{ID: "claude", Name: "Anthropic Claude Code", Ready: false, Cost: core.CostIncluded},
			{ID: "codex", Name: "OpenAI Codex / Work", Ready: false, Cost: core.CostScarce},
		},
	}
	view := runnerProviderCacheView{Providers: []core.RunnerProviderInfo{
		{ID: "workbench-runner", Name: "Workbench Cluster Runner", Ready: true, Cost: core.CostIncluded},
		{ID: "openclaw", Name: "OpenClaw", Ready: true, Cost: core.CostIncluded, Status: "CLI detected"},
		{ID: "claude", Name: "Anthropic Claude Code", Ready: false, Cost: core.CostIncluded, Status: "not detected"},
		{ID: "codex", Name: "OpenAI Codex / Work", Ready: true, Cost: core.CostScarce, Status: "connected"},
	}}

	got := applyRunnerProviderDashboard(snapshot, view)
	if got.ProviderReady != 3 || got.ProviderTotal != 4 {
		t.Fatalf("worker metric=%d/%d want 3/4: %+v", got.ProviderReady, got.ProviderTotal, got.Providers)
	}
	seen := map[string]DashboardProviderItem{}
	for _, provider := range got.Providers {
		seen[provider.ID] = provider
	}
	if !seen["workbench-runner"].Ready || !seen["runner:openclaw"].Ready || !seen["runner:codex"].Ready {
		t.Fatalf("runner-backed ready workers missing: %+v", got.Providers)
	}
	if _, exists := seen["codex"]; exists {
		t.Fatalf("local-PC Codex placeholder leaked into runner-backed dashboard: %+v", got.Providers)
	}
	if seen["runner:openclaw"].CurrentTask != "Runner · included-subscription" {
		t.Fatalf("runner detail=%q", seen["runner:openclaw"].CurrentTask)
	}
}

func TestDashboardProviderWindowAccountsForHiddenWorkers(t *testing.T) {
	visible, hidden := dashboardProviderWindow(8, dashboardProviderRowHeight*5+dashboardProviderFooterHeight)
	if visible != 5 || hidden != 3 {
		t.Fatalf("visible=%d hidden=%d want 5/3", visible, hidden)
	}
	visible, hidden = dashboardProviderWindow(4, dashboardProviderRowHeight*5)
	if visible != 4 || hidden != 0 {
		t.Fatalf("visible=%d hidden=%d want 4/0", visible, hidden)
	}
}
