package core

import (
	"strings"
	"testing"
)

func TestTelemetryPhaseIsTruncatedAtConfiguredBound(t *testing.T) {
	progress, ok := normalizeTaskTelemetry(WorkProgress{Kind: ProgressStages, Stage: 1, StageTotal: 2, Phase: strings.Repeat("x", maxTelemetryPhaseRunes+50)})
	if !ok || len([]rune(progress.Phase)) != maxTelemetryPhaseRunes {
		t.Fatalf("phase len=%d ok=%v", len([]rune(progress.Phase)), ok)
	}
}
