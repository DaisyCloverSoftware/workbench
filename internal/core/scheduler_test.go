package core

import (
	"testing"
	"time"
)

func TestSchedulerLaneCapacitiesSeparateExecutionPlanes(t *testing.T) {
	for _, lane := range []WorkLane{WorkLaneServerOps, WorkLaneCIBuilds, WorkLaneWindowsWorkstation, WorkLaneAIWorkers} {
		if got := schedulerLaneCapacity(lane); got != 1 {
			t.Fatalf("lane %q capacity=%d want 1", lane, got)
		}
	}
	if got := schedulerLaneCapacity(WorkLaneWaiting); got != 0 {
		t.Fatalf("waiting capacity=%d want 0", got)
	}
	if got := schedulerLaneCapacity(WorkLaneNeedsYou); got != 0 {
		t.Fatalf("needs-you capacity=%d want 0", got)
	}
}

func TestQueuePositionUsesPriorityBeforeFIFO(t *testing.T) {
	now := time.Date(2026, 8, 20, 15, 0, 0, 0, time.UTC)
	tasks := []Task{
		{ID: "normal", Status: TaskQueued, CreatedAt: now, Priority: PriorityNormal},
		{ID: "low", Status: TaskQueued, CreatedAt: now.Add(-time.Minute), Priority: PriorityLow},
		{ID: "critical", Status: TaskQueued, CreatedAt: now.Add(time.Minute), Priority: PriorityCritical},
		{ID: "high", Status: TaskQueued, CreatedAt: now.Add(2 * time.Minute), Priority: PriorityHigh},
	}
	positions := QueuePositions(tasks)
	if positions["critical"] != 1 || positions["high"] != 2 || positions["normal"] != 3 || positions["low"] != 4 {
		t.Fatalf("positions=%v", positions)
	}
}

func TestProgressFallsBackToIndeterminate(t *testing.T) {
	progress := TaskProgress(Task{Status: TaskRunning, Progress: WorkProgress{Kind: ProgressMeasured, Total: 0}})
	if progress.Kind != ProgressIndeterminate {
		t.Fatalf("progress kind=%q", progress.Kind)
	}
	if progress.Phase != "Implementing" {
		t.Fatalf("phase=%q", progress.Phase)
	}
}
