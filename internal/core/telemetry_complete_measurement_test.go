package core

import "testing"

func TestMeasuredTelemetryAllowsCurrentEqualTotal(t *testing.T) {
	progress, ok := normalizeTaskTelemetry(WorkProgress{Kind: ProgressMeasured, Current: 10, Total: 10, Unit: "files", Phase: "Verification complete"})
	if !ok || progress.Current != progress.Total {
		t.Fatalf("completed measurement=%#v ok=%v", progress, ok)
	}
}
