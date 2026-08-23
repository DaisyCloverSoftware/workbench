package core

import "testing"

func TestTelemetryPhaseBoundIsReasonable(t *testing.T) {
	if maxTelemetryPhaseRunes <= 0 || maxTelemetryPhaseRunes > 512 {
		t.Fatalf("unexpected telemetry phase bound %d", maxTelemetryPhaseRunes)
	}
}
