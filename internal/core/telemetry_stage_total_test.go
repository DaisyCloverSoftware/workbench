package core

import "testing"

func TestStageTelemetryRejectsUnboundedStageCounts(t *testing.T) {
	if progress, ok := normalizeTaskTelemetry(WorkProgress{Kind: ProgressStages, Stage: 1, StageTotal: maxTelemetryStages + 1, Phase: "Too many"}); ok {
		t.Fatalf("unbounded stages accepted: %#v", progress)
	}
}
