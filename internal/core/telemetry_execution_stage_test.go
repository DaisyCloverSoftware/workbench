package core

import "testing"

func TestRunningFallbackNeverReturnsGenericWorking(t *testing.T) {
	got := TaskProgress(Task{Status: TaskRunning, Progress: WorkProgress{Kind: ProgressIndeterminate, Phase: "Running"}})
	if got.Phase == "Running" || got.Phase == "Working" {
		t.Fatalf("generic running fallback=%#v", got)
	}
}
