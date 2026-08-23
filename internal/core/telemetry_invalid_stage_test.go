package core

import "testing"

func TestTelemetryRejectsNegativeStage(t *testing.T) {
	if progress, ok := normalizeTaskTelemetry(WorkProgress{Kind: ProgressStages, Stage: -1, StageTotal: 4, Phase: "Bad"}); ok {
		t.Fatalf("negative stage accepted: %#v", progress)
	}
}
