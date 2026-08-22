package core

import (
	"path/filepath"
	"testing"
	"time"
)

func TestSetTaskPriorityChangesNextSchedulerDispatch(t *testing.T) {
	now := time.Date(2026, 8, 22, 14, 0, 0, 0, time.UTC)
	store, err := NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	e := &Engine{
		store: store,
		state: State{Tasks: []Task{
			{ID: "busy", Status: TaskRunning, CreatedAt: now.Add(-time.Hour), UpdatedAt: now, ProviderID: "openclaw"},
			{ID: "first", Status: TaskQueued, CreatedAt: now.Add(-2 * time.Minute), UpdatedAt: now, Priority: PriorityNormal},
			{ID: "promoted", Status: TaskQueued, CreatedAt: now.Add(-time.Minute), UpdatedAt: now, Priority: PriorityLow},
		}},
		cancel:        map[string]context.CancelFunc{},
		schedulerWake: make(chan struct{}, 1),
	}
	if err := store.Save(e.state); err != nil {
		t.Fatal(err)
	}

	if err := e.SetTaskPriority("promoted", PriorityCritical); err != nil {
		t.Fatal(err)
	}
	positions := QueuePositions(e.State().Tasks)
	if positions["promoted"] != 1 || positions["first"] != 2 {
		t.Fatalf("positions after reprioritisation=%v", positions)
	}

	e.mu.Lock()
	e.state.Tasks[0].Status = TaskCompleted
	e.mu.Unlock()
	launch := e.schedulerDispatch()
	if len(launch) != 1 || launch[0] != "promoted" {
		t.Fatalf("next dispatch=%v want [promoted]", launch)
	}
	if task, ok := e.Task("promoted"); !ok || task.Status != TaskRouting {
		t.Fatalf("promoted task after dispatch=%#v ok=%v", task, ok)
	}
	if task, ok := e.Task("first"); !ok || task.Status != TaskQueued {
		t.Fatalf("first task should remain queued: %#v ok=%v", task, ok)
	}
}

func TestSetTaskPriorityRejectsNonQueuedTask(t *testing.T) {
	store, err := NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	e := &Engine{
		store: store,
		state: State{Tasks: []Task{{ID: "running", Status: TaskRunning}}},
		cancel:        map[string]context.CancelFunc{},
		schedulerWake: make(chan struct{}, 1),
	}
	if err := e.SetTaskPriority("running", PriorityCritical); err == nil {
		t.Fatal("expected reprioritising a running task to fail")
	}
}
