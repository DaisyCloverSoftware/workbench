package core

import "testing"

func TestMeasuredTelemetryRetainsOperatorReadablePhase(t *testing.T) {
	progress, ok := normalizeTaskTelemetry(WorkProgress{Kind: ProgressMeasured, Current: 3, Total: 6, Unit: "files", Phase: "Verifying source files"})
	if !ok || progress.Phase != "Verifying source files" {
		t.Fatalf("progress=%#v ok=%v", progress, ok)
	}
}
