package core

import (
	"path/filepath"
	"testing"
	"time"
)

func TestEngineTaskTelemetryPersistsAndNotifiesCanonicalState(t *testing.T) {
	store, err := NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	state := DefaultState()
	state.Tasks = []Task{{
		ID: "telemetry", Title: "Telemetry", Status: TaskRunning, Mode: TaskModeDevelopment,
		Priority: PriorityHigh, CreatedAt: now.Add(-time.Minute), UpdatedAt: now.Add(-10 * time.Second), StartedAt: &now,
		Progress: WorkProgress{Kind: ProgressStages, Phase: "Executing worker", Stage: 2, StageTotal: 4},
	}}
	if err := store.Save(state); err != nil {
		t.Fatal(err)
	}
	eng, err := NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	notified := make(chan struct{}, 1)
	eng.Subscribe(func() {
		select {
		case notified <- struct{}{}:
		default:
		}
	})
	eng.updateTaskTelemetry("telemetry", WorkProgress{Kind: ProgressMeasured, Current: 64, Total: 100, Unit: "files", Phase: "Verifying source files"})
	got, ok := eng.Task("telemetry")
	if !ok {
		t.Fatal("task missing")
	}
	if got.Progress.Kind != ProgressMeasured || got.Progress.Current != 64 || got.Progress.Total != 100 || got.Progress.Phase != "Verifying source files" {
		t.Fatalf("progress=%#v", got.Progress)
	}
	if !got.UpdatedAt.After(state.Tasks[0].UpdatedAt) {
		t.Fatalf("updated_at did not advance: %v", got.UpdatedAt)
	}
	select {
	case <-notified:
	case <-time.After(time.Second):
		t.Fatal("telemetry did not notify subscribers")
	}
	persisted, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(persisted.Tasks) != 1 || persisted.Tasks[0].Progress.Current != 64 {
		t.Fatalf("persisted=%#v", persisted.Tasks)
	}
}
