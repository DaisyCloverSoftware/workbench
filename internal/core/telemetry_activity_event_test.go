package core

import "testing"

func TestTelemetryPhaseSurvivesNormalizationAsMeaningfulActivity(t *testing.T) {
	progress, ok := normalizeTaskTelemetry(WorkProgress{Kind: ProgressMeasured, Current: 4, Total: 8, Phase: "Compiling package graph"})
	if !ok || progress.Phase != "Compiling package graph" {
		t.Fatalf("progress=%#v ok=%v", progress, ok)
	}
}
