package core

import (
	"strings"
	"testing"
)

func TestTelemetryUnitIsBounded(t *testing.T) {
	progress, ok := normalizeTaskTelemetry(WorkProgress{Kind: ProgressMeasured, Current: 1, Total: 2, Unit: strings.Repeat("u", 100), Phase: "Working"})
	if !ok {
		t.Fatal("valid measured telemetry rejected")
	}
	if len([]rune(progress.Unit)) > maxTelemetryUnitRunes {
		t.Fatalf("unit not bounded: %d", len([]rune(progress.Unit)))
	}
}
