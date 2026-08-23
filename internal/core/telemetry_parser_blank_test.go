package core

import "testing"

func TestHarnessProgressRejectsBlankPayload(t *testing.T) {
	if progress, ok := parseHarnessProgressLine("WORKBENCH_PROGRESS:   "); ok {
		t.Fatalf("blank telemetry accepted: %#v", progress)
	}
}
