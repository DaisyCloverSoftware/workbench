package core

import "testing"

func TestTelemetryRejectsSecretLookingPhase(t *testing.T) {
	if progress, ok := normalizeTaskTelemetry(WorkProgress{Kind: ProgressMeasured, Current: 1, Total: 2, Phase: "token=ghp_abcdefghijklmnopqrstuvwxyz123456"}); ok {
		t.Fatalf("secret-looking telemetry accepted: %#v", progress)
	}
}
