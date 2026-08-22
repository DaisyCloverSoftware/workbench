package desktop

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestDashboardOperationsSurfaceUsesOneCanonicalLiveSet(t *testing.T) {
	now := time.Date(2026, 8, 22, 14, 0, 0, 0, time.UTC)
	projectPath := t.TempDir()
	started := now.Add(-12 * time.Minute)
	dependencyStarted := now.Add(-20 * time.Minute)
	nextCheck := now.Add(2 * time.Minute)
	state := core.DefaultState()
	state.Projects = []core.Project{{ID: "project-1", Path: projectPath, Name: "Workbench", AddedAt: now, LastUsedAt: now}}
	state.ActiveProjectID = "project-1"
	state.ProjectPath = projectPath
	state.Tasks = []core.Task{
		{
			ID: "run", ProjectPath: projectPath, Title: "Implement dashboard", Status: core.TaskRunning,
			Priority: core.PriorityNormal, ProviderID: "openclaw", CreatedAt: now.Add(-15 * time.Minute), UpdatedAt: now,
			StartedAt: &started, Progress: core.WorkProgress{Kind: core.ProgressStages, Phase: "Tests", Stage: 2, StageTotal: 5},
			RouteReason: "included worker assigned",
		},
		{
			ID: "queued", ProjectPath: projectPath, Title: "Queued follow-up", Status: core.TaskQueued,
			Priority: core.PriorityHigh, CreatedAt: now.Add(-10 * time.Minute), UpdatedAt: now.Add(-9 * time.Minute),
			Progress: core.WorkProgress{Kind: core.ProgressIndeterminate, Phase: "Queued"},
		},
		{
			ID: "waiting", ProjectPath: projectPath, Title: "Wait for CI", Status: core.TaskWaitingDependency,
			CreatedAt: now.Add(-30 * time.Minute), UpdatedAt: now.Add(-time.Minute),
			Dependency: &core.TaskDependency{Kind: core.DependencyGitHubActions, Reason: "required CI", Repository: "DaisyCloverSoftware/workbench", RunID: 1234, State: "in_progress", StartedAt: dependencyStarted, NextCheckAt: nextCheck},
		},
		{
			ID: "need", ProjectPath: projectPath, Title: "Needs decision", Status: core.TaskNeedsAttention,
			CreatedAt: now.Add(-8 * time.Minute), UpdatedAt: now.Add(-2 * time.Minute), AttentionQuestion: "Approve the deployment target?",
		},
		{
			ID: "done", ProjectPath: projectPath, Title: "Completed task", Status: core.TaskCompleted,
			CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-5 * time.Minute),
			Review: &core.TaskReviewResult{Changed: true, Branch: "workbench/review", Commit: "1234567890abcdef"},
		},
		{
			ID: "failed", ProjectPath: projectPath, Title: "Failed task", Status: core.TaskFailed,
			CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-4 * time.Minute), Error: "build failed",
		},
	}
	store, err := core.NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(state); err != nil {
		t.Fatal(err)
	}
	eng, err := core.NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	remote := []core.RunnerChatActivityInfo{
		{ID: "remote-run", ProjectRef: "runner://workbench", Action: "run_safe_command", State: "running", UpdatedAt: now, Active: true, ActiveKnown: true},
		{ID: "remote-done", ProjectRef: "runner://workbench", Action: "run_safe_command", State: "completed", UpdatedAt: now.Add(-3 * time.Minute), ActiveKnown: true},
		{ID: "remote-failed", ProjectRef: "runner://workbench", Action: "run_safe_command", State: "failed", UpdatedAt: now.Add(-2 * time.Minute), ActiveKnown: true},
	}

	surface := buildDashboardOperationsSurface(eng, remote)
	if err := dashboardOperationsInvariantError(surface); err != nil {
		t.Fatal(err)
	}
	if surface.Live.Running != 2 || surface.Live.Queued != 1 || surface.Live.Waiting != 1 || surface.Live.NeedsHuman != 1 {
		t.Fatalf("live totals=%#v", surface.Live)
	}
	if len(surface.Live.Items) != 5 {
		t.Fatalf("live items=%d want 5", len(surface.Live.Items))
	}
	if len(surface.RecentOutcomes) != 4 {
		t.Fatalf("recent outcomes=%#v", surface.RecentOutcomes)
	}
	if detail := surface.Details["run"]; detail.Progress != "Tests · stage 2/5" || detail.Worker != "openclaw" {
		t.Fatalf("running detail=%#v", detail)
	}
	waiting := surface.Details["waiting"]
	if !strings.Contains(waiting.WaitReason, "GitHub Actions run 1234") || !strings.Contains(waiting.AutoContinuation, "check again") || waiting.WaitSince.IsZero() {
		t.Fatalf("waiting detail=%#v", waiting)
	}
	needs := surface.Details["need"]
	if !strings.Contains(needs.OwnerAction, "Approve the deployment target?") || !needs.LocalTask {
		t.Fatalf("needs-you detail=%#v", needs)
	}
	failed := surface.Details["failed"]
	if failed.Failure != "build failed" {
		t.Fatalf("failure detail=%#v", failed)
	}
	completed := surface.Details["done"]
	if !strings.Contains(completed.Reference, "workbench/review") || !strings.Contains(completed.Reference, "1234567890") {
		t.Fatalf("completed reference=%#v", completed)
	}
}

func TestDashboardOperationsPriorityControlsUseSchedulerPriorityScale(t *testing.T) {
	if got, ok := higherOperationsPriority(core.PriorityLow); !ok || got != core.PriorityNormal {
		t.Fatalf("raise low=(%v,%v)", got, ok)
	}
	if got, ok := higherOperationsPriority(core.PriorityNormal); !ok || got != core.PriorityHigh {
		t.Fatalf("raise normal=(%v,%v)", got, ok)
	}
	if got, ok := higherOperationsPriority(core.PriorityHigh); !ok || got != core.PriorityCritical {
		t.Fatalf("raise high=(%v,%v)", got, ok)
	}
	if _, ok := higherOperationsPriority(core.PriorityCritical); ok {
		t.Fatal("critical should not raise further")
	}
	if got, ok := lowerOperationsPriority(core.PriorityCritical); !ok || got != core.PriorityHigh {
		t.Fatalf("lower critical=(%v,%v)", got, ok)
	}
	if got, ok := lowerOperationsPriority(core.PriorityHigh); !ok || got != core.PriorityNormal {
		t.Fatalf("lower high=(%v,%v)", got, ok)
	}
	if got, ok := lowerOperationsPriority(core.PriorityNormal); !ok || got != core.PriorityLow {
		t.Fatalf("lower normal=(%v,%v)", got, ok)
	}
	if _, ok := lowerOperationsPriority(core.PriorityLow); ok {
		t.Fatal("low should not lower further")
	}
}

func TestDashboardOperationsIndeterminateRunningProgressDoesNotInventPercent(t *testing.T) {
	got := operationsProgressSummary(core.WorkProgress{Kind: core.ProgressIndeterminate, Phase: "Implementing"}, core.TaskRunning)
	if strings.Contains(got, "%") {
		t.Fatalf("indeterminate progress invented percent: %q", got)
	}
	if !strings.Contains(got, "Implementing") || !strings.Contains(got, "deterministic percentage unavailable") {
		t.Fatalf("indeterminate progress=%q", got)
	}
}
