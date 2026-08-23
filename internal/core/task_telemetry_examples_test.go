package core

import "testing"

func TestTelemetryExamplesRemainBoundedAndTruthful(t *testing.T) {
	if _, ok := normalizeTaskTelemetry(WorkProgress{Kind: ProgressMeasured, Current: 3, Total: 5, Unit: "checks", Phase: "Running checks"}); !ok {
		t.Fatal("valid measured telemetry rejected")
	}
	if _, ok := normalizeTaskTelemetry(WorkProgress{Kind: ProgressStages, Stage: 2, StageTotal: 4, Phase: "Executing"}); !ok {
		t.Fatal("valid stage telemetry rejected")
	}
}
