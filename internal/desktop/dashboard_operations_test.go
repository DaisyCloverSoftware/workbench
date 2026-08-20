package desktop

import (
	"testing"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestWorkItemElapsedUsesStartedAtWhenAvailable(t *testing.T) {
	created := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	started := created.Add(10 * time.Minute)
	item := core.WorkItem{CreatedAt: created, StartedAt: &started}
	if got := workItemElapsed(item, started.Add(17*time.Minute)); got != "17m" {
		t.Fatalf("elapsed=%q", got)
	}
}

func TestDependencySummaryNamesGitHubRun(t *testing.T) {
	task := core.Task{Dependency: &core.TaskDependency{Kind: core.DependencyGitHubActions, RunID: 12345}}
	if got := dependencySummary(task); got != "GitHub Actions run 12345" {
		t.Fatalf("summary=%q", got)
	}
}
