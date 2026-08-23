package core

import "testing"

func TestNormalizeTelemetryDoesNotAcceptIndeterminateProviderClaims(t *testing.T) {
	if progress, ok := normalizeTaskTelemetry(WorkProgress{Kind: ProgressIndeterminate, Phase: "Working"}); ok {
		t.Fatalf("indeterminate provider claim accepted as telemetry: %#v", progress)
	}
}
