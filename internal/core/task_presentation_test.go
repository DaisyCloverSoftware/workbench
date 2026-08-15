package core

import "testing"

func TestPresentTaskMakesAutonomousNextActionExplicit(t *testing.T) {
	p := PresentTask(Task{Status: TaskRunning, ProviderID: "claude"})
	if p.StatusLabel != "Working" {
		t.Fatalf("status label = %q", p.StatusLabel)
	}
	if p.NeedsHuman || p.Terminal {
		t.Fatalf("running task flags are wrong: %#v", p)
	}
	if p.NextAction == "" {
		t.Fatal("running task has no next action")
	}
}

func TestPresentTaskExtractsWorkbenchReviewRef(t *testing.T) {
	p := PresentTask(Task{
		Status: TaskCompleted,
		Output: "Worker report.\n\nWorkbench published review branch workbench/task-123 at 0123456789abcdef. The same prepared commit was already present on the publication target.",
	})
	if p.StatusLabel != "Ready" || !p.Terminal || !p.Published {
		t.Fatalf("presentation = %#v", p)
	}
	if p.ReviewBranch != "workbench/task-123" {
		t.Fatalf("review branch = %q", p.ReviewBranch)
	}
	if p.ReviewCommit != "0123456789abcdef" {
		t.Fatalf("review commit = %q", p.ReviewCommit)
	}
}

func TestPresentTaskNeedsHumanOnlyAtAttentionBoundary(t *testing.T) {
	p := PresentTask(Task{Status: TaskNeedsAttention, AttentionQuestion: "Choose A or B"})
	if !p.NeedsHuman || p.Terminal {
		t.Fatalf("attention presentation = %#v", p)
	}
	if p.StatusLabel != "Needs you" {
		t.Fatalf("status label = %q", p.StatusLabel)
	}
}

func TestSummarizeTasks(t *testing.T) {
	s := SummarizeTasks([]Task{
		{Status: TaskQueued},
		{Status: TaskRunning},
		{Status: TaskNeedsAttention},
		{Status: TaskCompleted},
		{Status: TaskFailed},
		{Status: TaskCancelled},
	})
	if s.Active != 2 || s.NeedsHuman != 1 || s.Completed != 1 || s.Failed != 1 {
		t.Fatalf("summary = %#v", s)
	}
}
