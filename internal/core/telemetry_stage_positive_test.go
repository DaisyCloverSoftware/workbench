package core

import "testing"

func TestStageTelemetryRequiresPositiveStageTotal(t *testing.T) {
	if progress, ok := normalizeTaskTelemetry(WorkProgress{Kind: ProgressStages, Stage: 1, StageTotal: 0, Phase: "Working"}); ok {
		t.Fatalf("stage telemetry without total accepted: %#v", progress)
	}
}
