package core

import "testing"

func TestTelemetryRejectsUnknownProgressKinds(t *testing.T) {
	if progress, ok := normalizeTaskTelemetry(WorkProgress{Kind: ProgressKind("guess"), Phase: "Working"}); ok {
		t.Fatalf("unknown progress kind accepted: %#v", progress)
	}
}
