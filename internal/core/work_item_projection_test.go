package core

import (
	"testing"
	"time"
)

func TestWorkItemFromTaskMapsOperationsPriorityAndTruthfulProgress(t *testing.T) {
	now := time.Date(2026, 8, 20, 8, 30, 0, 0, time.UTC)
	project := Project{ID: "project-rum", Name: "RUM", Path: "/tmp/rum", DefaultPriority: WorkPriorityCritical}
	task := Task{
		ID:          "task-rum-deploy",
		Title:       "Deploy public alpha",
		ProjectPath: project.Path,
		Mode:        TaskModeOperations,
		Status:      TaskRunning,
		ProviderID:  "workbench-runner",
		CreatedAt:   now.Add(-time.Minute),
		UpdatedAt:   now,
	}

	item := WorkItemFromTask(task, project)
	if item.Priority != WorkPriorityCritical {
		t.Fatalf("priority = %q, want critical", item.Priority)
	}
	if item.Location.Lane != WorkLaneServerOperations {
		t.Fatalf("lane = %q, want server operations", item.Location.Lane)
	}
	if item.Progress.Kind != WorkProgressIndeterminate {
		t.Fatalf("running task progress = %q, want indeterminate", item.Progress.Kind)
	}
	if _, ok := item.Progress.Percent(); ok {
		t.Fatal("running task must not fabricate a percentage")
	}
	if item.CanReprioritize {
		t.Fatal("already-running work should not claim it can be reordered")
	}
	if !item.CanCancel {
		t.Fatal("running task should remain cancellable")
	}
}

func TestWorkItemFromTaskMapsGitHubDependencyToCILane(t *testing.T) {
	task := Task{
		ID:     "task-ci",
		Title:  "Wait for build",
		Status: TaskWaitingDependency,
		Dependency: &TaskDependency{
			Kind:   DependencyGitHubActions,
			RunID:  12345,
			Reason: "Windows build still running",
		},
	}
	item := WorkItemFromTask(task, Project{Name: "Workbench"})
	if item.State != WorkItemWaiting || item.Location.Lane != WorkLaneCIBuilds {
		t.Fatalf("CI projection = %#v", item)
	}
	if item.Blocker != "Windows build still running" {
		t.Fatalf("blocker = %q", item.Blocker)
	}
	if !item.CanReprioritize {
		t.Fatal("waiting work should allow a priority change for its next scheduling point")
	}
}

func TestAssignLaneQueuePositionsKeepsQueuesIndependent(t *testing.T) {
	base := time.Date(2026, 8, 20, 8, 0, 0, 0, time.UTC)
	items := []WorkItem{
		{ID: "ops-normal", State: WorkItemQueued, Priority: WorkPriorityNormal, CreatedAt: base, Location: WorkLocation{Lane: WorkLaneServerOperations}},
		{ID: "ops-critical", State: WorkItemQueued, Priority: WorkPriorityCritical, CreatedAt: base.Add(time.Minute), Location: WorkLocation{Lane: WorkLaneServerOperations}},
		{ID: "ai-high", State: WorkItemQueued, Priority: WorkPriorityHigh, CreatedAt: base.Add(2 * time.Minute), Location: WorkLocation{Lane: WorkLaneAIWorkers}},
		{ID: "ops-running", State: WorkItemRunning, Priority: WorkPriorityCritical, CreatedAt: base.Add(-time.Minute), Location: WorkLocation{Lane: WorkLaneServerOperations}},
	}

	got := AssignLaneQueuePositions(items)
	positions := map[string]int{}
	for _, item := range got {
		positions[item.ID] = item.QueuePosition
	}
	if positions["ops-critical"] != 1 || positions["ops-normal"] != 2 {
		t.Fatalf("server queue positions = %#v", positions)
	}
	if positions["ai-high"] != 1 {
		t.Fatalf("AI queue should start independently at 1, got %#v", positions)
	}
	if positions["ops-running"] != 0 {
		t.Fatalf("running work is not queued, got position %d", positions["ops-running"])
	}
}
