package core

import "testing"

func TestUnmeasuredRunningWorkAlwaysHasStagePosition(t *testing.T) {
	got := TaskProgress(Task{Status: TaskRunning, Progress: WorkProgress{Kind: ProgressIndeterminate}})
	if got.Kind != ProgressStages || got.Stage <= 0 || got.StageTotal <= 0 {
		t.Fatalf("unmeasured running work has no stage position: %#v", got)
	}
}
