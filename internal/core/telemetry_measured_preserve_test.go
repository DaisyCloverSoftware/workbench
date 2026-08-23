package core

import "testing"

func TestTaskProgressPreservesProviderSuppliedMeasuredTelemetry(t *testing.T) {
	task := Task{Status: TaskRunning, Progress: WorkProgress{Kind: ProgressMeasured, Current: 7, Total: 10, Unit: "files", Phase: "Verifying files"}}
	got := TaskProgress(task)
	if got.Kind != ProgressMeasured || got.Current != 7 || got.Total != 10 || got.Phase != "Verifying files" {
		t.Fatalf("provider measured telemetry was replaced: %#v", got)
	}
}
