package core

import "testing"

func TestMeasuredTelemetryMayOmitUnitWhenCurrentTotalAreDefensible(t *testing.T) {
	progress, ok := normalizeTaskTelemetry(WorkProgress{Kind: ProgressMeasured, Current: 2, Total: 4, Phase: "Executing checks"})
	if !ok || progress.Current != 2 || progress.Total != 4 {
		t.Fatalf("progress=%#v ok=%v", progress, ok)
	}
}
