package core

import "testing"

func TestMeasuredTelemetryKeepsRealUnit(t *testing.T) {
	progress, ok := normalizeTaskTelemetry(WorkProgress{Kind: ProgressMeasured, Current: 8, Total: 12, Unit: "files", Phase: "Hashing source"})
	if !ok || progress.Unit != "files" {
		t.Fatalf("progress=%#v ok=%v", progress, ok)
	}
}
