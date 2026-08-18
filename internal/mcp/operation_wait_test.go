package mcp

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func operationWaitTestEngine(t *testing.T, task core.Task) *core.Engine {
	t.Helper()
	store, err := core.NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	state := core.DefaultState()
	state.Tasks = []core.Task{task}
	if err := store.Save(state); err != nil {
		t.Fatal(err)
	}
	eng, err := core.NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	return eng
}

func TestAwaitOperationReturnsCompletedImmediately(t *testing.T) {
	task := core.Task{ID: "task-complete", Mode: core.TaskModeOperations, Status: core.TaskCompleted, Output: "verified"}
	eng := operationWaitTestEngine(t, task)

	started := time.Now()
	got, err := awaitOperation(context.Background(), eng, task.ID, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if got.WaitTimedOut || got.Task.Status != core.TaskCompleted || got.Task.Output != "verified" {
		t.Fatalf("unexpected wait result: %+v", got)
	}
	if time.Since(started) > 200*time.Millisecond {
		t.Fatalf("terminal operation should return immediately; elapsed=%s", time.Since(started))
	}
}

func TestAwaitOperationReturnsNeedsAttentionImmediately(t *testing.T) {
	task := core.Task{ID: "task-attention", Mode: core.TaskModeOperations, Status: core.TaskNeedsAttention, AttentionQuestion: "Approve production restart?"}
	eng := operationWaitTestEngine(t, task)

	got, err := awaitOperation(context.Background(), eng, task.ID, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if got.WaitTimedOut || got.Task.Status != core.TaskNeedsAttention || got.Task.AttentionQuestion == "" {
		t.Fatalf("unexpected attention result: %+v", got)
	}
}

func TestAwaitOperationTimesOutWithoutChangingDurableTask(t *testing.T) {
	task := core.Task{ID: "task-running", Mode: core.TaskModeOperations, Status: core.TaskRunning}
	eng := operationWaitTestEngine(t, task)

	got, err := awaitOperation(context.Background(), eng, task.ID, 20*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if !got.WaitTimedOut || got.Task.Status != core.TaskRunning {
		t.Fatalf("unexpected timeout result: %+v", got)
	}
	persisted, ok := eng.Task(task.ID)
	if !ok || persisted.Status != core.TaskRunning {
		t.Fatalf("bounded wait mutated durable task: %+v ok=%t", persisted, ok)
	}
}

func TestAwaitOperationRejectsDevelopmentTask(t *testing.T) {
	task := core.Task{ID: "task-development", Mode: core.TaskModeDevelopment, Status: core.TaskRunning}
	eng := operationWaitTestEngine(t, task)

	if _, err := awaitOperation(context.Background(), eng, task.ID, 20*time.Millisecond); err == nil {
		t.Fatal("await_operation accepted a development task")
	}
}

func TestOperationWaitDurationIsBounded(t *testing.T) {
	if got := operationWaitDuration(0); got != defaultOperationWait {
		t.Fatalf("default wait=%s want %s", got, defaultOperationWait)
	}
	if got := operationWaitDuration(99999); got != maxOperationWait {
		t.Fatalf("max wait=%s want %s", got, maxOperationWait)
	}
}
