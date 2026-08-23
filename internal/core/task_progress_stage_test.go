package core

import "testing"

func TestRoutingProgressUsesExecutionLifecycleStage(t *testing.T) {
	got := TaskProgress(Task{Status: TaskRouting, Progress: WorkProgress{Kind: ProgressIndeterminate, Phase: "Selecting executor"}})
	if got.Kind != ProgressStages || got.Stage != 1 || got.StageTotal != 4 || got.Phase != "Selecting executor" {
		t.Fatalf("routing progress=%#v", got)
	}
}
