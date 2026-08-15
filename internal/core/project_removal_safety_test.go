package core

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestTaskBlocksProjectRemovalOnlyForUnfinishedStatuses(t *testing.T) {
	cases := map[TaskStatus]bool{
		TaskQueued:         true,
		TaskRouting:        true,
		TaskRunning:        true,
		TaskWaitingRetry:   true,
		TaskNeedsAttention: true,
		TaskCompleted:      false,
		TaskFailed:         false,
		TaskCancelled:      false,
	}
	for status, want := range cases {
		if got := taskBlocksProjectRemoval(status); got != want {
			t.Fatalf("taskBlocksProjectRemoval(%q)=%t want %t", status, got, want)
		}
	}
}

func TestRemoveProjectRejectsUnfinishedWorkAndPreservesProject(t *testing.T) {
	store, err := NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	eng, err := NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	project, err := eng.SelectProject(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	eng.mu.Lock()
	eng.state.Tasks = []Task{{
		ID:                "needs-human",
		ProjectPath:       project.Path,
		Status:            TaskNeedsAttention,
		AttentionQuestion: "Choose A or B",
	}}
	eng.mu.Unlock()

	err = eng.RemoveProject(project.ID)
	if err == nil || !strings.Contains(err.Error(), "unfinished Workbench task") {
		t.Fatalf("expected unfinished-work refusal, got %v", err)
	}
	projects := eng.Projects()
	if len(projects) != 1 || projects[0].ID != project.ID {
		t.Fatalf("failed removal changed project registry: %#v", projects)
	}
	active, ok := eng.ActiveProject()
	if !ok || active.ID != project.ID {
		t.Fatalf("failed removal changed active project: %#v ok=%t", active, ok)
	}
	if task, ok := eng.Task("needs-human"); !ok || task.Status != TaskNeedsAttention {
		t.Fatalf("failed removal changed unfinished task: %#v ok=%t", task, ok)
	}
}

func TestRemoveProjectAllowsTerminalHistoryWithoutDeletingTasks(t *testing.T) {
	store, err := NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	eng, err := NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	project, err := eng.SelectProject(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	eng.mu.Lock()
	eng.state.Tasks = []Task{
		{ID: "completed", ProjectPath: project.Path, Status: TaskCompleted},
		{ID: "failed", ProjectPath: project.Path, Status: TaskFailed},
		{ID: "cancelled", ProjectPath: project.Path, Status: TaskCancelled},
	}
	eng.mu.Unlock()

	if err := eng.RemoveProject(project.ID); err != nil {
		t.Fatal(err)
	}
	if projects := eng.Projects(); len(projects) != 0 {
		t.Fatalf("terminal-only project remained registered: %#v", projects)
	}
	st := eng.State()
	if len(st.Tasks) != 3 {
		t.Fatalf("project removal deleted terminal task history: %#v", st.Tasks)
	}

	reloaded, err := NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	if projects := reloaded.Projects(); len(projects) != 0 {
		t.Fatalf("terminal history resurrected removed project after restart: %#v", projects)
	}
	if len(reloaded.State().Tasks) != 3 {
		t.Fatal("terminal task history was not durable after project removal")
	}
}
