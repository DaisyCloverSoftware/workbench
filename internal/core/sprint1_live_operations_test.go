package core

import (
	"context"
	"path/filepath"
	"testing"
)

func TestDelegateOperationQueuesThroughScheduler(t *testing.T) {
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
	task, err := e.DelegateOperation("desktop-ui", "verify the host", project)
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != TaskQueued {
		t.Fatalf("delegated operation status=%q want queued", task.Status)
	}
	stored, ok := e.Task(task.ID)
	if !ok || stored.Status != TaskQueued {
		t.Fatalf("stored operation=%#v ok=%v; direct execution bypassed the scheduler", stored, ok)
	}
	select {
	case <-e.schedulerWake:
		// Expected: scheduler, not DelegateOperation, owns queued -> routing.
	default:
		t.Fatal("delegated operation did not wake the scheduler")
	}
}

func TestTaskProgressUsesMeaningfulRunningStage(t *testing.T) {
	development := Task{Status: TaskRunning, Progress: WorkProgress{Kind: ProgressIndeterminate, Phase: "Running"}}
	dev := TaskProgress(development)
	if dev.Kind != ProgressStages || dev.Stage != 2 || dev.StageTotal != 4 || dev.Phase != "Executing worker" {
		t.Fatalf("development running progress=%#v", dev)
	}
	operation := development
	operation.Mode = TaskModeOperations
	ops := TaskProgress(operation)
	if ops.Kind != ProgressStages || ops.Stage != 2 || ops.StageTotal != 4 || ops.Phase != "Executing operation" {
		t.Fatalf("operations running progress=%#v", ops)
	}
	measured := Task{Status: TaskRunning, Progress: WorkProgress{Kind: ProgressMeasured, Phase: "Tests", Current: 3, Total: 10, Unit: "checks"}}
	got := TaskProgress(measured)
	if got.Phase != "Tests" || got.Current != 3 || got.Total != 10 {
		t.Fatalf("deterministic progress was overwritten: %#v", got)
	}
}
