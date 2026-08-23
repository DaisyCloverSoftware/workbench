package core

import "testing"

func TestHarnessProgressPrefixMustBeExact(t *testing.T) {
	if progress, ok := parseHarnessProgressLine(`workbench_progress: {"kind":"stages","stage":1,"stage_total":2,"phase":"Working"}`); ok {
		t.Fatalf("noncanonical progress prefix accepted: %#v", progress)
	}
}
