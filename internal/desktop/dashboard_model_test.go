package desktop

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestBuildDashboardSnapshotUsesOnlyDurableWorkbenchFacts(t *testing.T) {
	store, err := core.NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	eng, err := core.NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	pinned, err := eng.SelectProject(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := eng.SetProjectPinned(pinned.ID, true); err != nil {
		t.Fatal(err)
	}
	other, err := eng.SelectProject(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	retryAt := now.Add(5 * time.Minute)
	st := eng.State()
	st.Preferences.OpenClawSSHHost = "runner.example"
	st.Tasks = []core.Task{
		{ID: "attention", ProjectPath: pinned.Path, Title: "Choose release behaviour", Status: core.TaskNeedsAttention, ProviderID: "claude", UpdatedAt: now, AttentionQuestion: "Choose A or B"},
		{ID: "retry", ProjectPath: pinned.Path, Title: "Retry provider", Status: core.TaskWaitingRetry, UpdatedAt: now.Add(-time.Minute), RetryAt: &retryAt},
		{ID: "done", ProjectPath: other.Path, Title: "Finished task", Status: core.TaskCompleted, UpdatedAt: now.Add(-2 * time.Minute)},
		{ID: "failed", ProjectPath: other.Path, Title: "Failed task", Status: core.TaskFailed, UpdatedAt: now.Add(-3 * time.Minute)},
	}
	if err := store.Save(st); err != nil {
		t.Fatal(err)
	}
	eng, err = core.NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}

	d := BuildDashboardSnapshot(eng)
	if d.ProjectCount != 2 || !d.RunnerConfigured {
		t.Fatalf("project/runner facts wrong: %#v", d)
	}
	if d.Summary.Active != 1 || d.Summary.NeedsHuman != 1 || d.Summary.Completed != 1 || d.Summary.Failed != 1 {
		t.Fatalf("task summary wrong: %#v", d.Summary)
	}
	if d.SuccessRate != 50 {
		t.Fatalf("success rate=%d want 50", d.SuccessRate)
	}
	if len(d.ActiveTasks) != 2 || d.ActiveTasks[0].TaskID != "attention" || d.ActiveTasks[1].TaskID != "retry" {
		t.Fatalf("active tasks wrong: %#v", d.ActiveTasks)
	}
	if !d.ActiveTasks[0].NeedsHuman || d.ActiveTasks[1].RetryAt == nil || !d.ActiveTasks[1].RetryAt.Equal(retryAt) {
		t.Fatalf("attention/retry facts lost: %#v", d.ActiveTasks)
	}
	if len(d.RecentActivity) != 4 || d.RecentActivity[0].TaskID != "attention" {
		t.Fatalf("recent activity wrong: %#v", d.RecentActivity)
	}
	if len(d.Projects) != 2 || d.Projects[0].ID != pinned.ID || !d.Projects[0].Pinned {
		t.Fatalf("project ordering/summary wrong: %#v", d.Projects)
	}
}

func TestDashboardActiveStatusIncludesWaitingAndHumanAttention(t *testing.T) {
	for _, status := range []core.TaskStatus{core.TaskQueued, core.TaskRouting, core.TaskRunning, core.TaskWaitingRetry, core.TaskNeedsAttention} {
		if !isDashboardActiveStatus(status) {
			t.Fatalf("status %q should be active on dashboard", status)
		}
	}
	for _, status := range []core.TaskStatus{core.TaskCompleted, core.TaskFailed, core.TaskCancelled} {
		if isDashboardActiveStatus(status) {
			t.Fatalf("terminal status %q appeared active", status)
		}
	}
}
