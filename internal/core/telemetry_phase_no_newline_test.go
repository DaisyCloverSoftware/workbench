package core

import "testing"

func TestTelemetryPhaseNewlinesBecomeSingleReadableLine(t *testing.T) {
	progress, ok := normalizeTaskTelemetry(WorkProgress{Kind: ProgressStages, Stage: 1, StageTotal: 2, Phase: "Preparing\nworkspace"})
	if !ok || progress.Phase != "Preparing workspace" {
		t.Fatalf("progress=%#v ok=%v", progress, ok)
	}
}
