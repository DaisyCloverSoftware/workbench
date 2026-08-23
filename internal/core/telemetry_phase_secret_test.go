package core

import "testing"

func TestTelemetryRejectsSecretLookingUnit(t *testing.T) {
	if progress, ok := normalizeTaskTelemetry(WorkProgress{Kind: ProgressMeasured, Current: 1, Total: 2, Unit: "ghp_abcdefghijklmnopqrstuvwxyz123456", Phase: "Working"}); ok {
		t.Fatalf("secret-looking unit accepted: %#v", progress)
	}
}
