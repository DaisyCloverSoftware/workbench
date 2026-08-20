package core

import "testing"

func TestValidateWorkProgressMeasuredAndStages(t *testing.T) {
	measured, err := ValidateWorkProgress(WorkProgress{Kind: WorkProgressMeasured, Current: 17, Total: 120, Unit: "frames"})
	if err != nil {
		t.Fatal(err)
	}
	if percent, ok := measured.Percent(); !ok || percent != 14 {
		t.Fatalf("measured percent = (%d, %v)", percent, ok)
	}
	stages, err := ValidateWorkProgress(WorkProgress{Kind: WorkProgressStages, Stage: 3, StageTotal: 5, StageName: "Rollout"})
	if err != nil {
		t.Fatal(err)
	}
	if percent, ok := stages.Percent(); !ok || percent != 60 {
		t.Fatalf("stage percent = (%d, %v)", percent, ok)
	}
}

func TestValidateWorkProgressRejectsImpossibleOrMultilineValues(t *testing.T) {
	bad := []WorkProgress{
		{Kind: WorkProgressMeasured, Current: 5, Total: 4},
		{Kind: WorkProgressStages, Stage: 6, StageTotal: 5},
		{Kind: WorkProgressIndeterminate, StageName: "phase\nsecret"},
		{Kind: "made_up"},
	}
	for _, progress := range bad {
		if _, err := ValidateWorkProgress(progress); err == nil {
			t.Fatalf("expected invalid progress to fail: %#v", progress)
		}
	}
}

func TestUpdateTaskProgressPersistsAndProjectsMeasuredValue(t *testing.T) {
	store, err := NewStoreAt(t.TempDir() + "/state.json")
	if err != nil {
		t.Fatal(err)
	}
	eng, err := NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	eng.mu.Lock()
	eng.state.Tasks = append(eng.state.Tasks, Task{ID: "task-progress", Title: "Render", Status: TaskRunning})
	eng.mu.Unlock()
	if err := eng.UpdateTaskProgress("task-progress", WorkProgress{Kind: WorkProgressMeasured, Current: 17, Total: 120, Unit: "frames"}); err != nil {
		t.Fatal(err)
	}
	task, ok := eng.Task("task-progress")
	if !ok {
		t.Fatal("task missing")
	}
	item := WorkItemFromTask(task, Project{})
	if percent, ok := item.Progress.Percent(); !ok || percent != 14 {
		t.Fatalf("projected progress = %#v", item.Progress)
	}
}

func TestUpdateTaskProgressRejectsTerminalTasks(t *testing.T) {
	store, err := NewStoreAt(t.TempDir() + "/state.json")
	if err != nil {
		t.Fatal(err)
	}
	eng, err := NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	eng.mu.Lock()
	eng.state.Tasks = append(eng.state.Tasks, Task{ID: "task-done", Status: TaskCompleted})
	eng.mu.Unlock()
	if err := eng.UpdateTaskProgress("task-done", WorkProgress{Kind: WorkProgressIndeterminate, StageName: "Working"}); err == nil {
		t.Fatal("terminal task progress should not be mutable")
	}
}
