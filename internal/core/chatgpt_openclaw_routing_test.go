package core

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDelegateOperationRejectsImplicitOpenClawForEveryOrigin(t *testing.T) {
	for _, origin := range []string{"chatgpt-mcp", "workbench-ui", "relay", "manual"} {
		e := &Engine{}
		_, err := e.DelegateOperation(origin, RelayOperationsIntentPrefix+" restart a service", "")
		if err == nil || !strings.Contains(err.Error(), "explicit owner authorization naming OpenClaw is required") {
			t.Fatalf("origin %q implicit OpenClaw delegation was not refused: %v", origin, err)
		}
	}
}

func TestDelegateOperationPersistsExplicitOwnerOpenClawAuthorization(t *testing.T) {
	store, err := NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	e := &Engine{
		store:         store,
		state:         DefaultState(),
		cancel:        map[string]context.CancelFunc{},
		schedulerWake: make(chan struct{}, 1),
	}
	project := t.TempDir()
	task, err := e.DelegateOperation("chatgpt-mcp", OpenClawExplicitAuthorizationPrefix+" investigate the runtime", project)
	if err != nil {
		t.Fatal(err)
	}
	if !task.OpenClawOwnerAuthorized || task.Mode != TaskModeOperations {
		t.Fatalf("explicit OpenClaw authorization was not persisted on task: %#v", task)
	}
	stored, ok := e.Task(task.ID)
	if !ok || !stored.OpenClawOwnerAuthorized {
		t.Fatalf("durable task lost explicit OpenClaw authorization: %#v ok=%v", stored, ok)
	}
}

func TestSchedulerSealsExplicitOwnerAuthorizationFromCompatibilityTask(t *testing.T) {
	store, err := NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	e := &Engine{
		store:         store,
		state:         DefaultState(),
		cancel:        map[string]context.CancelFunc{},
		schedulerWake: make(chan struct{}, 1),
	}
	project := t.TempDir()
	intent := OpenClawExplicitAuthorizationPrefix + " " + RelayOperationsIntentPrefix + " inspect the runtime"
	task, err := e.Delegate("desktop", intent, project)
	if err != nil {
		t.Fatal(err)
	}
	if task.OpenClawOwnerAuthorized {
		t.Fatalf("compatibility entry point should not synthesize durable authorization before scheduler sealing: %#v", task)
	}
	launch := e.schedulerDispatch()
	if len(launch) != 1 || launch[0] != task.ID {
		t.Fatalf("scheduler launch=%#v, want explicitly authorized task %q", launch, task.ID)
	}
	stored, ok := e.Task(task.ID)
	if !ok {
		t.Fatal("scheduled task disappeared")
	}
	if !stored.OpenClawOwnerAuthorized || stored.Mode != TaskModeOperations {
		t.Fatalf("scheduler did not persist explicit OpenClaw owner authorization: %#v", stored)
	}
	if strings.Contains(stored.Intent, OpenClawExplicitAuthorizationPrefix) {
		t.Fatalf("authorization transport marker leaked into operational objective: %q", stored.Intent)
	}
	if !strings.Contains(stored.Intent, RelayOperationsIntentPrefix) {
		t.Fatalf("operations routing metadata was unexpectedly removed: %q", stored.Intent)
	}
}

func TestOperationsMarkerAloneNeverBecomesOwnerAuthorization(t *testing.T) {
	task := Task{Intent: RelayOperationsIntentPrefix + " kubectl apply", Mode: TaskModeOperations}
	if applyExplicitOwnerOpenClawAuthorization(&task) {
		t.Fatalf("ordinary operations routing metadata became OpenClaw authorization: %#v", task)
	}
	if task.OpenClawOwnerAuthorized {
		t.Fatalf("ordinary operations routing metadata persisted OpenClaw authorization: %#v", task)
	}
}

func TestRouteCandidatesNeverSelectOpenClawWithoutDurableOwnerAuthorization(t *testing.T) {
	providers := []Provider{
		{ID: "openclaw", Name: "OpenClaw", Installed: true, Authenticated: true, CanWrite: true, Command: "openclaw", Cost: CostIncluded},
		{ID: "claude", Name: "Claude", Installed: true, Authenticated: true, CanWrite: true, Command: "claude", Cost: CostIncluded},
	}

	unauthorizedOperation := Task{Mode: TaskModeOperations, ProjectPath: t.TempDir(), Intent: "restart service"}
	if got := routeCandidates(providers, Preferences{}, unauthorizedOperation); len(got) != 0 {
		t.Fatalf("unauthorized Operations task received provider candidates: %#v", got)
	}

	authorizedOperation := unauthorizedOperation
	authorizedOperation.OpenClawOwnerAuthorized = true
	got := routeCandidates(providers, Preferences{}, authorizedOperation)
	if len(got) != 1 || got[0].ID != "openclaw" {
		t.Fatalf("explicitly authorized operation candidates=%#v, want OpenClaw only", got)
	}

	development := Task{Mode: TaskModeDevelopment, ProjectPath: t.TempDir(), Intent: "implement feature"}
	got = routeCandidates(providers, Preferences{}, development)
	if len(got) != 1 || got[0].ID != "claude" {
		t.Fatalf("ordinary development routing selected OpenClaw: %#v", got)
	}
}

func TestRetireLegacyChatGPTOperationTasksCancelsAllUnauthorizedOpenClawWork(t *testing.T) {
	now := time.Date(2026, 8, 20, 3, 20, 0, 0, time.UTC)
	retryAt := now.Add(5 * time.Minute)
	st := State{Tasks: []Task{
		{
			ID:           "task-legacy-running",
			Origin:       "chatgpt-mcp",
			Mode:         TaskModeOperations,
			Status:       TaskRunning,
			ProviderID:   "openclaw",
			RouteReason:  "legacy",
			ConsumesWork: false,
		},
		{
			ID:         "task-legacy-retry",
			Origin:     "chatgpt-mcp",
			Mode:       TaskModeOperations,
			Status:     TaskWaitingRetry,
			ProviderID: "openclaw",
			RetryAt:    &retryAt,
		},
		{
			ID:         "task-old-manual-without-proof",
			Origin:     "workbench-ui",
			Mode:       TaskModeOperations,
			Status:     TaskRunning,
			ProviderID: "openclaw",
		},
		{
			ID:                      "task-explicit-owner-authorized",
			Origin:                  "workbench-ui",
			Mode:                    TaskModeOperations,
			OpenClawOwnerAuthorized: true,
			Status:                  TaskRunning,
			ProviderID:              "openclaw",
		},
		{
			ID:         "task-legacy-complete",
			Origin:     "chatgpt-mcp",
			Mode:       TaskModeOperations,
			Status:     TaskCompleted,
			FinishedAt: timePointer(now.Add(-time.Minute)),
		},
	}}

	if !retireLegacyChatGPTOperationTasks(&st, now) {
		t.Fatal("expected unauthorized legacy OpenClaw operations to be retired")
	}
	for _, index := range []int{0, 1, 2} {
		got := st.Tasks[index]
		if got.Status != TaskCancelled || got.ProviderID != "" || got.RouteReason != "" || got.RetryAt != nil || got.FinishedAt == nil {
			t.Fatalf("unauthorized legacy operation was not fully retired: %#v", got)
		}
		if len(got.Attempts) == 0 || !strings.Contains(got.Attempts[len(got.Attempts)-1], "no durable explicit owner authorization") {
			t.Fatalf("legacy task retirement reason missing: %#v", got.Attempts)
		}
	}
	authorized := st.Tasks[3]
	if authorized.Status != TaskRunning || authorized.ProviderID != "openclaw" || !authorized.OpenClawOwnerAuthorized {
		t.Fatalf("explicitly owner-authorized OpenClaw operation was incorrectly retired: %#v", authorized)
	}
	if st.Tasks[4].Status != TaskCompleted {
		t.Fatalf("terminal task was changed: %#v", st.Tasks[4])
	}
}
