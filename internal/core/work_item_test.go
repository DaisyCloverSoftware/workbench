package core

import (
	"testing"
	"time"
)

func TestQueuePositionsRespectPriorityThenFIFOWithinLane(t *testing.T) {
	now := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	tasks := []Task{
		{ID: "normal-first", CreatedAt: now, Status: TaskQueued, Mode: TaskModeDevelopment, Priority: PriorityNormal},
		{ID: "high", CreatedAt: now.Add(time.Second), Status: TaskQueued, Mode: TaskModeDevelopment, Priority: PriorityHigh},
		{ID: "critical", CreatedAt: now.Add(2 * time.Second), Status: TaskQueued, Mode: TaskModeDevelopment, Priority: PriorityCritical},
		{ID: "ops", CreatedAt: now.Add(3 * time.Second), Status: TaskQueued, Mode: TaskModeOperations, Priority: PriorityNormal},
	}
	positions := QueuePositions(tasks)
	if positions["critical"] != 1 || positions["high"] != 2 || positions["normal-first"] != 3 {
		t.Fatalf("development positions critical=%d high=%d normal=%d", positions["critical"], positions["high"], positions["normal-first"])
	}
	if positions["ops"] != 1 {
		t.Fatalf("operations lane position=%d want 1", positions["ops"])
	}
}

func TestHistoricalZeroPriorityIsNormal(t *testing.T) {
	if got := DefaultTaskPriority(Task{}); got != PriorityNormal {
		t.Fatalf("historical task priority=%v want normal", got)
	}
}

func TestTaskLaneSeparatesWaitsAndHumanAttention(t *testing.T) {
	if got := TaskLane(Task{Status: TaskNeedsAttention}); got != WorkLaneNeedsYou {
		t.Fatalf("needs attention lane=%q", got)
	}
	if got := TaskLane(Task{Status: TaskWaitingDependency, Dependency: &TaskDependency{Kind: DependencyGitHubActions}}); got != WorkLaneWaiting {
		t.Fatalf("dependency wait lane=%q", got)
	}
	if got := TaskLane(Task{Status: TaskRunning, Mode: TaskModeOperations}); got != WorkLaneServerOps {
		t.Fatalf("operations lane=%q", got)
	}
}

func TestTaskProgressDoesNotInventPercentages(t *testing.T) {
	got := TaskProgress(Task{Status: TaskRunning})
	if got.Kind != ProgressIndeterminate || got.Total != 0 {
		t.Fatalf("progress=%+v", got)
	}
}
