package core

import (
	"strings"
	"testing"
)

func TestHarnessProgressRejectsOversizedPayload(t *testing.T) {
	line := "WORKBENCH_PROGRESS: " + strings.Repeat("x", maxHarnessProgressBytes+1)
	if progress, ok := parseHarnessProgressLine(line); ok {
		t.Fatalf("oversized telemetry accepted: %#v", progress)
	}
}
