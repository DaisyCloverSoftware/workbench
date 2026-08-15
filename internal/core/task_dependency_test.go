package core

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestParseDeferredGitHubActionsIntent(t *testing.T) {
	now := time.Date(2026, 8, 15, 21, 0, 0, 0, time.UTC)
	intent := `WORKBENCH_WAIT_GITHUB_ACTIONS: {"repository":"example/workbench","run_id":123456}
Continue the release task after CI and fix any in-scope failure.`
	dependency, continuation, matched, err := parseDeferredGitHubActionsIntent(intent, now)
	if err != nil || !matched {
		t.Fatalf("matched=%t err=%v", matched, err)
	}
	if dependency == nil || dependency.Kind != DependencyGitHubActions || dependency.Repository != "example/workbench" || dependency.RunID != 123456 {
		t.Fatalf("unexpected dependency: %#v", dependency)
	}
	if !dependency.NextCheckAt.Equal(now.Add(initialDependencyCheckDelay)) {
		t.Fatalf("next check=%v want %v", dependency.NextCheckAt, now.Add(initialDependencyCheckDelay))
	}
	if continuation != "Continue the release task after CI and fix any in-scope failure." {
		t.Fatalf("unexpected continuation %q", continuation)
	}

	if _, _, matched, err := parseDeferredGitHubActionsIntent("ordinary coding intent", now); err != nil || matched {
		t.Fatalf("ordinary intent matched=%t err=%v", matched, err)
	}
	for _, bad := range []string{
		`WORKBENCH_WAIT_GITHUB_ACTIONS: {"repository":"not a slug","run_id":123}` + "\ncontinue",
		`WORKBENCH_WAIT_GITHUB_ACTIONS: {"repository":"example/workbench","run_id":0}` + "\ncontinue",
		`WORKBENCH_WAIT_GITHUB_ACTIONS: {"repository":"example/workbench","run_id":123,"extra":true}` + "\ncontinue",
		`WORKBENCH_WAIT_GITHUB_ACTIONS: {"repository":"example/workbench","run_id":123}`,
	} {
		if _, _, matched, err := parseDeferredGitHubActionsIntent(bad, now); !matched || err == nil {
			t.Fatalf("invalid intent matched=%t err=%v: %q", matched, err, bad)
		}
	}
}

func TestDependencyPollDelayBacksOffWithoutHammering(t *testing.T) {
	states := []string{"queued", "in_progress", "probe_unavailable"}
	for _, state := range states {
		previous := time.Duration(0)
		for check := 1; check <= 12; check++ {
			delay := dependencyPollDelay(check, state)
			if delay < 20*time.Second {
				t.Fatalf("%s check %d polls too aggressively: %v", state, check, delay)
			}
			if delay < previous {
				t.Fatalf("%s delay regressed from %v to %v", state, previous, delay)
			}
			previous = delay
		}
	}
	if got := dependencyPollDelay(20, "queued"); got > 2*time.Minute {
		t.Fatalf("queued CI became too sluggish: %v", got)
	}
	if got := dependencyPollDelay(20, "probe_unavailable"); got != 10*time.Minute {
		t.Fatalf("unavailable dependency backoff=%v want 10m", got)
	}
}

func TestGitHubActionsProbeUsesOneBoundedStatusLookup(t *testing.T) {
	var gotArgs []string
	run := func(_ context.Context, args ...string) ([]byte, error) {
		gotArgs = append([]string(nil), args...)
		return []byte(`{"databaseId":987654,"status":"completed","conclusion":"success","workflowName":"build"}`), nil
	}
	observation, err := probeGitHubActionsRunWithRunner(context.Background(), "example/workbench", 987654, run)
	if err != nil {
		t.Fatal(err)
	}
	wantArgs := []string{"run", "view", "987654", "--repo", "example/workbench", "--json", "databaseId,status,conclusion,workflowName"}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("args=%q want %q", gotArgs, wantArgs)
	}
	if !observation.Completed || observation.Status != "completed" || observation.Conclusion != "success" || observation.WorkflowName != "build" {
		t.Fatalf("unexpected observation: %#v", observation)
	}

	if _, err := probeGitHubActionsRunWithRunner(context.Background(), "example/workbench", 987654, func(context.Context, ...string) ([]byte, error) {
		return nil, errors.New("offline")
	}); err == nil {
		t.Fatal("expected probe failure")
	}
}

func TestDeferredDependencyCreatesIdleDurableTaskAndDeduplicates(t *testing.T) {
	store, err := NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	st := DefaultState()
	if err := store.Save(st); err != nil {
		t.Fatal(err)
	}
	e := &Engine{store: store, state: st, cancel: map[string]context.CancelFunc{}}
	intent := `WORKBENCH_WAIT_GITHUB_ACTIONS: {"repository":"example/workbench","run_id":777}
Continue after CI.`

	first, matched, err := e.tryDelegateDeferredDependency("test", intent, project)
	if err != nil || !matched {
		t.Fatalf("matched=%t err=%v", matched, err)
	}
	if first.Status != TaskWaitingDependency || first.ProviderID != "" || first.StartedAt != nil || first.ConsumesWork || first.Dependency == nil {
		t.Fatalf("dependency task reserved a worker or was not durable: %#v", first)
	}
	second, matched, err := e.tryDelegateDeferredDependency("test", intent, project)
	if err != nil || !matched || second.ID != first.ID {
		t.Fatalf("duplicate watch was not reused: first=%s second=%s matched=%t err=%v", first.ID, second.ID, matched, err)
	}
	if got := len(e.State().Tasks); got != 1 {
		t.Fatalf("duplicate wait created %d tasks", got)
	}
}

func TestRecoverWaitingDependencyRearmsAndRejectsCorruptState(t *testing.T) {
	now := time.Date(2026, 8, 15, 21, 30, 0, 0, time.UTC)
	future := now.Add(2 * time.Minute)
	st := DefaultState()
	st.Tasks = []Task{
		{ID: "future", Status: TaskWaitingDependency, Dependency: &TaskDependency{Kind: DependencyGitHubActions, Repository: "example/workbench", RunID: 1, NextCheckAt: future}},
		{ID: "overdue", Status: TaskWaitingDependency, Dependency: &TaskDependency{Kind: DependencyGitHubActions, Repository: "example/workbench", RunID: 2, NextCheckAt: now.Add(-time.Minute)}},
		{ID: "corrupt", Status: TaskWaitingDependency},
	}
	schedules, changed := recoverWaitingDependencyTasks(&st, now)
	if !changed {
		t.Fatal("overdue/corrupt dependency recovery did not change state")
	}
	if len(schedules) != 2 {
		t.Fatalf("got %d schedules want 2", len(schedules))
	}
	if !schedules[0].CheckAt.Equal(future) {
		t.Fatalf("future deadline changed: %v", schedules[0].CheckAt)
	}
	if !schedules[1].CheckAt.Equal(now.Add(time.Second)) {
		t.Fatalf("overdue check was not safely rearmed: %v", schedules[1].CheckAt)
	}
	if st.Tasks[2].Status != TaskFailed || st.Tasks[2].FinishedAt == nil {
		t.Fatalf("corrupt dependency was not failed safely: %#v", st.Tasks[2])
	}
}

func TestPresentWaitingDependencyExplainsBackgroundContinuation(t *testing.T) {
	next := time.Date(2026, 8, 15, 22, 0, 0, 0, time.UTC)
	p := PresentTask(Task{
		Status:     TaskWaitingDependency,
		Dependency: &TaskDependency{Kind: DependencyGitHubActions, State: "queued", NextCheckAt: next},
	})
	if p.StatusLabel != "Waiting on dependency" || p.NeedsHuman || p.Terminal {
		t.Fatalf("waiting dependency presentation=%#v", p)
	}
	if p.DependencyNextCheckAt == nil || !p.DependencyNextCheckAt.Equal(next) {
		t.Fatal(("dependency check deadline missing: %#v", p))
	}
	if p.DependencyKind != DependencyGitHubActions || p.DependencyState != "queued" {
		t.Fatalf("dependency presentation lost state: %#v", p)
	}
	if got := SummarizeTasks([]Task{{Status: TaskWaitingDependency}}); got.Active != 1 {
		t.Fatalf("waiting dependency not counted active: %#v", got)
	}
}

func TestCloneStateDeepCopiesDependency(t *testing.T) {
	st := DefaultState()
	st.Tasks = []Task{{ID: "task", Dependency: &TaskDependency{Kind: DependencyGitHubActions, State: "queued"}}}
	clone := cloneState(st)
	clone.Tasks[0].Dependency.State = "completed"
	if st.Tasks[0].Dependency.State != "queued" {
		t.Fatal("cloneState shared mutable dependency metadata")
	}
}
