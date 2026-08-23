package core

import "testing"

func TestInvalidTelemetryIsIgnoredBeforeEngineMutation(t *testing.T) {
	if progress, ok := normalizeTaskTelemetry(WorkProgress{Kind: ProgressMeasured, Current: 5, Total: 4, Phase: "Invalid"}); ok {
		t.Fatalf("invalid telemetry accepted: %#v", progress)
	}
}
