package core

import "testing"

func TestTaskProgressPreservesProviderSuppliedStageTelemetry(t *testing.T) {
	task := Task{Status: TaskRunning, Progress: WorkProgress{Kind: ProgressStages, Stage: 3, StageTotal: 5, Phase: "Verifying integration"}}
	got := TaskProgress(task)
	if got.Stage != 3 || got.StageTotal != 5 || got.Phase != "Verifying integration" {
		t.Fatalf("provider stage telemetry was replaced: %#v", got)
	}
}
