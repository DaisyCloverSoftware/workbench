package core

import "testing"

func TestMeasuredTelemetryUsesIntegerWorkCounts(t *testing.T) {
	progress, ok := normalizeTaskTelemetry(WorkProgress{Kind: ProgressMeasured, Current: 64, Total: 100, Unit: "files", Phase: "Verifying"})
	if !ok || progress.Current != 64 || progress.Total != 100 {
		t.Fatalf("progress=%#v ok=%v", progress, ok)
	}
}
