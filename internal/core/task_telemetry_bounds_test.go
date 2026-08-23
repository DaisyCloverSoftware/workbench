package core

import (
	"strings"
	"testing"
)

func TestTelemetryTextIsBoundedBeforePersistence(t *testing.T) {
	phase := strings.Repeat("phase ", 100)
	progress, ok := normalizeTaskTelemetry(WorkProgress{Kind: ProgressStages, Stage: 1, StageTotal: 2, Phase: phase})
	if !ok {
		t.Fatal("bounded valid telemetry rejected")
	}
	if len([]rune(progress.Phase)) > maxTelemetryPhaseRunes {
		t.Fatalf("phase not bounded: %d", len([]rune(progress.Phase)))
	}
}
