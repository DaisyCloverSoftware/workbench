package desktop

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestOperationsDashboardIncludesRemoteRelayWork(t *testing.T) {
	store, err := core.NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	eng, err := core.NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	remote := []core.RunnerChatActivityInfo{
		{ID: "task_12345678", ProjectRef: "runner://rum", Action: "delegate_task", State: "running", UpdatedAt: now, Active: true, ActiveKnown: true},
		{ID: "windows_12345678", Action: "run_windows_unreal_smoke", State: "running", UpdatedAt: now, Active: true, ActiveKnown: true},
		{ID: "wait_12345678", ProjectRef: "runner://workbench", Action: "delegate_task", State: "waiting", UpdatedAt: now, Active: true, ActiveKnown: true},
		{ID: "done_12345678", ProjectRef: "runner://garage", Action: "delegate_task", State: "completed", UpdatedAt: now, Active: false, ActiveKnown: true},
	}

	got := buildDashboardOperationsSnapshot(eng, remote)
	if got.Running != 2 || got.Waiting != 1 || got.Queued != 0 || got.NeedsHuman != 0 {
		t.Fatalf("summary=%#v", got)
	}
	if len(got.ByLane[core.WorkLaneAIWorkers]) != 1 || got.ByLane[core.WorkLaneAIWorkers][0].ProjectName != "rum" {
		t.Fatalf("AI lane=%#v", got.ByLane[core.WorkLaneAIWorkers])
	}
	if len(got.ByLane[core.WorkLaneWindowsWorkstation]) != 1 || got.ByLane[core.WorkLaneWindowsWorkstation][0].Title != "Unreal smoke" {
		t.Fatalf("Windows lane=%#v", got.ByLane[core.WorkLaneWindowsWorkstation])
	}
	if len(got.ByLane[core.WorkLaneWaiting]) != 1 || got.ByLane[core.WorkLaneWaiting][0].ProjectName != "workbench" {
		t.Fatalf("Waiting lane=%#v", got.ByLane[core.WorkLaneWaiting])
	}
	if len(got.Items) != 3 {
		t.Fatalf("items=%#v", got.Items)
	}
}

func TestOperationsDashboardPriorityOrderMatchesScheduler(t *testing.T) {
	items := []core.WorkItem{
		{ID: "normal", Priority: core.PriorityNormal, Lane: core.WorkLaneAIWorkers},
		{ID: "critical", Priority: core.PriorityCritical, Lane: core.WorkLaneAIWorkers},
		{ID: "high", Priority: core.PriorityHigh, Lane: core.WorkLaneAIWorkers},
		{ID: "low", Priority: core.PriorityLow, Lane: core.WorkLaneAIWorkers},
	}
	for _, item := range items {
		if dashboardPriorityRank(item.Priority) < 0 {
			t.Fatal("invalid priority rank")
		}
	}
	if !(dashboardPriorityRank(core.PriorityCritical) < dashboardPriorityRank(core.PriorityHigh) &&
		dashboardPriorityRank(core.PriorityHigh) < dashboardPriorityRank(core.PriorityNormal) &&
		dashboardPriorityRank(core.PriorityNormal) < dashboardPriorityRank(core.PriorityLow)) {
		t.Fatal("dashboard priority order diverges from scheduler")
	}
}
