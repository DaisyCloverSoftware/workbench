package core

import (
	"path/filepath"
	"testing"
	"time"
)

func engineWithArchiveTask(t *testing.T, status TaskStatus) (*Engine, *Store, string) {
	t.Helper()
	store, err := NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	state := DefaultState()
	state.Tasks = []Task{{
		ID:          "archive-task-12345678",
		CreatedAt:   now.Add(-time.Hour),
		UpdatedAt:   now,
		Origin:      "test",
		Title:       "Archive contract",
		Intent:      "prove reversible task history",
		ProjectPath: filepath.Join(t.TempDir(), "project"),
		Status:      status,
		Output:      "durable result",
		Attempts:    []string{"worker: completed"},
	}}
	if err := store.Save(state); err != nil {
		t.Fatal(err)
	}
	eng, err := NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	return eng, store, state.Tasks[0].ID
}

func TestSetTaskArchivedPreservesDurableRecordAndRestores(t *testing.T) {
	eng, store, id := engineWithArchiveTask(t, TaskCompleted)
	before, ok := eng.Task(id)
	if !ok {
		t.Fatal("fixture task missing")
	}
	if err := eng.SetTaskArchived(id, true); err != nil {
		t.Fatal(err)
	}
	task, ok := eng.Task(id)
	if !ok || !task.Archived || task.Output != "durable result" || len(task.Attempts) != 1 {
		t.Fatalf("archive mutated durable task record: %+v", task)
	}
	if !task.UpdatedAt.Equal(before.UpdatedAt) {
		t.Fatalf("archive rewrote task chronology: before=%s after=%s", before.UpdatedAt, task.UpdatedAt)
	}
	if len(eng.State().Tasks) != 1 {
		t.Fatalf("archive deleted task record: %+v", eng.State().Tasks)
	}

	reloaded, err := NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	persisted, ok := reloaded.Task(id)
	if !ok || !persisted.Archived || persisted.Output != "durable result" {
		t.Fatalf("archived state did not persist: %+v", persisted)
	}
	if err := reloaded.SetTaskArchived(id, false); err != nil {
		t.Fatal(err)
	}
	restored, ok := reloaded.Task(id)
	if !ok || restored.Archived || restored.Output != "durable result" {
		t.Fatalf("restore did not preserve task record: %+v", restored)
	}
	if !restored.UpdatedAt.Equal(before.UpdatedAt) {
		t.Fatalf("restore rewrote task chronology: before=%s after=%s", before.UpdatedAt, restored.UpdatedAt)
	}
}

func TestTaskArchiveAcceptsOnlyTerminalStatuses(t *testing.T) {
	for _, status := range []TaskStatus{TaskCompleted, TaskFailed, TaskCancelled} {
		eng, _, id := engineWithArchiveTask(t, status)
		if !TaskCanArchive(status) {
			t.Fatalf("terminal status %q should be archivable", status)
		}
		if err := eng.SetTaskArchived(id, true); err != nil {
			t.Fatalf("archive %q: %v", status, err)
		}
	}
	for _, status := range []TaskStatus{TaskQueued, TaskRouting, TaskRunning, TaskWaitingRetry, TaskWaitingDependency, TaskNeedsAttention} {
		eng, _, id := engineWithArchiveTask(t, status)
		if TaskCanArchive(status) {
			t.Fatalf("active status %q must not be archivable", status)
		}
		if err := eng.SetTaskArchived(id, true); err == nil {
			t.Fatalf("expected archive of %q to fail closed", status)
		}
		task, _ := eng.Task(id)
		if task.Archived {
			t.Fatalf("rejected active task %q was hidden", status)
		}
	}
}
