package core

import "testing"

func TestMeasuredTelemetryWithoutTotalIsRejected(t *testing.T) {
	if progress, ok := normalizeTaskTelemetry(WorkProgress{Kind: ProgressMeasured, Current: 1, Phase: "Working"}); ok {
		t.Fatalf("measurement without total accepted: %#v", progress)
	}
}
