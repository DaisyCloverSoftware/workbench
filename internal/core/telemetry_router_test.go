package core

import "testing"

func TestRoutingStageDoesNotExposePercentage(t *testing.T) {
	got := TaskProgress(Task{Status: TaskRouting, Progress: WorkProgress{Kind: ProgressIndeterminate}})
	if got.Kind != ProgressStages || got.Stage != 1 || got.StageTotal != 4 {
		t.Fatalf("routing progress=%#v", got)
	}
}
