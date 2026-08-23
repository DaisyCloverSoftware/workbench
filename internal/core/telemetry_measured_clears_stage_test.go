package core

import "testing"

func TestMeasuredTelemetryClearsStageFields(t *testing.T) {
	progress, ok := normalizeTaskTelemetry(WorkProgress{Kind: ProgressMeasured, Current: 5, Total: 10, Stage: 3, StageTotal: 4, Phase: "Working"})
	if !ok {
		t.Fatal("valid measured telemetry rejected")
	}
	if progress.Stage != 0 || progress.StageTotal != 0 {
		t.Fatalf("measured telemetry retained stage fields: %#v", progress)
	}
}
