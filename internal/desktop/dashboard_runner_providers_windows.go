//go:build windows

package desktop

import (
	"sort"
	"strings"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

const (
	dashboardProviderRowHeight    = 34
	dashboardProviderFooterHeight = 20
)

// applyRunnerProviderDashboard replaces misleading local-PC worker health with
// the actual autonomous capacity available behind the configured cluster
// runner. The runner gateway remains visible as the control plane; nested
// workbench-runner and OpenClaw model-choice rows are omitted because they are
// not additional execution workers.
func applyRunnerProviderDashboard(snapshot DashboardSnapshot, view runnerProviderCacheView) DashboardSnapshot {
	if len(view.Providers) == 0 {
		return snapshot
	}

	items := make([]DashboardProviderItem, 0, len(view.Providers)+1)
	for _, existing := range snapshot.Providers {
		if existing.ID == "workbench-runner" {
			items = append(items, existing)
			break
		}
	}

	for _, provider := range view.Providers {
		if provider.ID == "workbench-runner" {
			continue
		}
		if _, isModel := core.RunnerCloudModelRefFromProviderID(provider.ID); isModel {
			continue
		}
		name := strings.TrimSpace(provider.Name)
		if provider.ID == core.RunnerConnectionProviderID {
			name = "Runner inventory"
		}
		if name == "" {
			name = provider.ID
		}
		status := strings.TrimSpace(provider.Status)
		items = append(items, DashboardProviderItem{
			ID:          "runner:" + provider.ID,
			Name:        name,
			Status:      status,
			Cost:        provider.Cost,
			Ready:       provider.Ready,
			Cooling:     strings.Contains(strings.ToLower(status), "cooldown"),
			CurrentTask: runnerProviderDashboardDetail(provider),
		})
	}

	if len(items) == 0 {
		return snapshot
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Ready != items[j].Ready {
			return items[i].Ready
		}
		if items[i].Cooling != items[j].Cooling {
			return !items[i].Cooling
		}
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})

	snapshot.Providers = items
	snapshot.ProviderReady = 0
	snapshot.ProviderTotal = len(items)
	for _, provider := range items {
		if provider.Ready {
			snapshot.ProviderReady++
		}
	}
	return snapshot
}

func runnerProviderDashboardDetail(provider core.RunnerProviderInfo) string {
	if provider.ID == core.RunnerConnectionProviderID {
		if status := strings.TrimSpace(provider.Status); status != "" {
			return status
		}
		return "cluster connection"
	}
	cost := strings.TrimSpace(string(provider.Cost))
	if cost == "" {
		return "Runner"
	}
	return "Runner · " + cost
}

func dashboardProviderWindow(providerCount, availableHeight int) (visible, hidden int) {
	if providerCount <= 0 || availableHeight <= 0 {
		return 0, 0
	}
	visible = availableHeight / dashboardProviderRowHeight
	if visible >= providerCount {
		return providerCount, 0
	}
	usable := availableHeight - dashboardProviderFooterHeight
	visible = usable / dashboardProviderRowHeight
	if visible < 1 {
		visible = 1
	}
	if visible > providerCount {
		visible = providerCount
	}
	return visible, providerCount - visible
}
