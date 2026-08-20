package core

import (
	"testing"
	"time"
)

func TestQueuePositionsAreFIFOWithinLane(t *testing.T) {
	now := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	tasks := []Task{
		{ID: "later", CreatedAt: now.Add(time.Second), Status: TaskQueued, Mode: TaskModeDevelopment},
		{ID: "first", CreatedAt: now, Status: TaskQueued, Mode: TaskModeDevelopment},
		{ID: "ops", CreatedAt: now.Add(2 * time.Second), Status: TaskQueued, Mode: TaskModeOperations},
	}
	positions := QueuePositions(tasks)
	if positions["first"] != 1 || positions["later"] != 2 {
		t.Fatalf("development queue positions = first:%d later:%d", positions["first"], positions["later"])
	}
	if positions["ops"] != 1 {
		t.Fatalf("operations lane position=%d want 1", positions["ops"])
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
