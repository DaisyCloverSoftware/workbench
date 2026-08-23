package core

import "testing"

func TestMeasuredTelemetryAllowsZeroCurrentWithRealTotal(t *testing.T) {
	progress, ok := normalizeTaskTelemetry(WorkProgress{Kind: ProgressMeasured, Current: 0, Total: 10, Unit: "files", Phase: "Starting verification"})
	if !ok || progress.Current != 0 || progress.Total != 10 {
		t.Fatalf("zero-current measured progress=%#v ok=%v", progress, ok)
	}
}
