package core

import "testing"

func TestTelemetryRequiresMeaningfulPhase(t *testing.T) {
	if progress, ok := normalizeTaskTelemetry(WorkProgress{Kind: ProgressMeasured, Current: 1, Total: 2}); ok {
		t.Fatalf("phase-less telemetry accepted: %#v", progress)
	}
}
