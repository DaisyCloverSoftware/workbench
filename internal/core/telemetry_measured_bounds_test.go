package core

import "testing"

func TestMeasuredTelemetryRequiresDefensibleDenominator(t *testing.T) {
	if progress, ok := normalizeTaskTelemetry(WorkProgress{Kind: ProgressMeasured, Current: 1, Total: -1, Phase: "Bad"}); ok {
		t.Fatalf("invalid denominator accepted: %#v", progress)
	}
}
