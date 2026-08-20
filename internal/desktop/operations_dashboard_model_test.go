package desktop

import (
	"testing"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestSummarizeOperationsDashboardBuildsStableLaneCounts(t *testing.T) {
	items := []core.WorkItem{
		{ID: "ops-running", State: core.WorkItemRunning, Location: core.WorkLocation{Lane: core.WorkLaneServerOperations}},
		{ID: "ops-queued", State: core.WorkItemQueued, Location: core.WorkLocation{Lane: core.WorkLaneServerOperations}, QueuePosition: 1},
		{ID: "ci-wait", State: core.WorkItemWaiting, Location: core.WorkLocation{Lane: core.WorkLaneCIBuilds}},
		{ID: "human", State: core.WorkItemNeedsAttention, Location: core.WorkLocation{Lane: core.WorkLaneNeedsHuman}},
	}

	snapshot := summarizeOperationsDashboard(items)
	if snapshot.Running != 1 || snapshot.Queued != 1 || snapshot.Waiting != 1 || snapshot.NeedsHuman != 1 {
		t.Fatalf("totals = running %d queued %d waiting %d needs-human %d", snapshot.Running, snapshot.Queued, snapshot.Waiting, snapshot.NeedsHuman)
	}
	if len(snapshot.Lanes) != 6 {
		t.Fatalf("lane count = %d, want 6 stable lanes", len(snapshot.Lanes))
	}
	if snapshot.Lanes[0].Lane != core.WorkLaneServerOperations || snapshot.Lanes[0].Running != 1 || snapshot.Lanes[0].Queued != 1 {
		t.Fatalf("server lane = %#v", snapshot.Lanes[0])
	}
	if snapshot.Lanes[1].Lane != core.WorkLaneCIBuilds || snapshot.Lanes[1].Waiting != 1 {
		t.Fatalf("CI lane = %#v", snapshot.Lanes[1])
	}
	if snapshot.Lanes[2].Lane != core.WorkLaneWindowsHost {
		t.Fatalf("empty Windows lane should stay visible, got %#v", snapshot.Lanes[2])
	}
}

func TestSummarizeOperationsDashboardUnknownLaneFallsBackToWaiting(t *testing.T) {
	snapshot := summarizeOperationsDashboard([]core.WorkItem{{ID: "unknown", State: core.WorkItemWaiting}})
	if snapshot.Waiting != 1 || len(snapshot.Lanes[4].Items) != 1 {
		t.Fatalf("unknown lane should remain visible under Waiting: %#v", snapshot)
	}
}
