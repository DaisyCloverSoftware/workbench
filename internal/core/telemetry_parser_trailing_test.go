package core

import "testing"

func TestHarnessProgressRejectsTrailingJSON(t *testing.T) {
	if progress, ok := parseHarnessProgressLine(`WORKBENCH_PROGRESS: {"kind":"stages","stage":1,"stage_total":2,"phase":"Working"} {"extra":true}`); ok {
		t.Fatalf("trailing telemetry JSON accepted: %#v", progress)
	}
}
