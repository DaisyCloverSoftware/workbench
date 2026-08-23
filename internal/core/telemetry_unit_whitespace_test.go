package core

import "testing"

func TestTelemetryUnitWhitespaceIsNormalized(t *testing.T) {
	progress, ok := normalizeTaskTelemetry(WorkProgress{Kind: ProgressMeasured, Current: 1, Total: 2, Unit: " source   files ", Phase: "Working"})
	if !ok || progress.Unit != "source files" {
		t.Fatalf("progress=%#v ok=%v", progress, ok)
	}
}
