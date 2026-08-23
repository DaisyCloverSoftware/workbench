package core

import "testing"

func TestTelemetryPhaseWhitespaceIsNormalized(t *testing.T) {
	progress, ok := normalizeTaskTelemetry(WorkProgress{Kind: ProgressStages, Stage: 1, StageTotal: 2, Phase: "  Preparing   workspace  "})
	if !ok || progress.Phase != "Preparing workspace" {
		t.Fatalf("progress=%#v ok=%v", progress, ok)
	}
}
