package core

import "testing"

func TestHarnessProgressPrefixConstantIsReserved(t *testing.T) {
	if harnessProgressPrefix != "WORKBENCH_PROGRESS:" {
		t.Fatalf("unexpected progress prefix %q", harnessProgressPrefix)
	}
}
