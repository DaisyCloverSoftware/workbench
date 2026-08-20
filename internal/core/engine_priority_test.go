package core

import "testing"

func TestEffectiveTaskPriorityInheritance(t *testing.T) {
	project := Project{DefaultPriority: WorkPriorityHigh}
	if got := EffectiveTaskPriority(Task{}, project); got != WorkPriorityHigh {
		t.Fatalf("inherited priority = %q, want high", got)
	}
	if got := EffectiveTaskPriority(Task{Priority: WorkPriorityCritical}, project); got != WorkPriorityCritical {
		t.Fatalf("task override = %q, want critical", got)
	}
	if got := EffectiveTaskPriority(Task{}, Project{}); got != WorkPriorityNormal {
		t.Fatalf("legacy project default = %q, want normal", got)
	}
}

func TestSetProjectPriorityPersists(t *testing.T) {
	store, err := NewStoreAt(t.TempDir() + "/state.json")
	if err != nil {
		t.Fatal(err)
	}
	eng, err := NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	projectPath := t.TempDir()
	project, err := eng.SelectProject(projectPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := eng.SetProjectPriority(project.ID, WorkPriorityCritical); err != nil {
		t.Fatal(err)
	}

	reloaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.Projects) != 1 || reloaded.Projects[0].DefaultPriority != WorkPriorityCritical {
		t.Fatalf("persisted projects = %#v", reloaded.Projects)
	}
}

func TestSetTaskPriorityOverrideAndClear(t *testing.T) {
	store, err := NewStoreAt(t.TempDir() + "/state.json")
	if err != nil {
		t.Fatal(err)
	}
	eng, err := NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	eng.mu.Lock()
	eng.state.Tasks = append(eng.state.Tasks, Task{ID: "task_priority_test", Title: "Priority test", Status: TaskQueued})
	eng.mu.Unlock()

	if err := eng.SetTaskPriority("task_priority_test", WorkPriorityLow); err != nil {
		t.Fatal(err)
	}
	got, ok := eng.Task("task_priority_test")
	if !ok || got.Priority != WorkPriorityLow {
		t.Fatalf("task after override = %#v", got)
	}
	if err := eng.SetTaskPriority("task_priority_test", ""); err != nil {
		t.Fatal(err)
	}
	got, ok = eng.Task("task_priority_test")
	if !ok || got.Priority != "" {
		t.Fatalf("task after clearing override = %#v", got)
	}
}

func TestPriorityControlsRejectUnknownValues(t *testing.T) {
	if IsWorkPriority("urgent-ish") {
		t.Fatal("unknown priority should not be accepted")
	}
	if got := NormalizeWorkPriority("urgent-ish"); got != WorkPriorityNormal {
		t.Fatalf("unknown projection priority = %q, want safe normal", got)
	}
}
