package core

import "testing"

func TestStageTelemetryRetainsOperatorReadablePhase(t *testing.T) {
	progress, ok := normalizeTaskTelemetry(WorkProgress{Kind: ProgressStages, Stage: 2, StageTotal: 4, Phase: "Executing checks"})
	if !ok || progress.Phase != "Executing checks" {
		t.Fatalf("progress=%#v ok=%v", progress, ok)
	}
}
