package core

import "testing"

func TestHarnessProgressRejectsUnknownJSONFields(t *testing.T) {
	if progress, ok := parseHarnessProgressLine(`WORKBENCH_PROGRESS: {"kind":"measured","current":1,"total":2,"phase":"Working","guess":50}`); ok {
		t.Fatalf("unknown telemetry field accepted: %#v", progress)
	}
}
