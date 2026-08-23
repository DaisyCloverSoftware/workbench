package core

import "testing"

func TestNormalizedTelemetryKeepsPhaseAlongsideProgressValues(t *testing.T) {
	progress, ok := normalizeTaskTelemetry(WorkProgress{Kind: ProgressMeasured, Current: 4, Total: 8, Unit: "files", Phase: "Reading dependency graph"})
	if !ok || progress.Phase != "Reading dependency graph" || progress.Current != 4 || progress.Total != 8 {
		t.Fatalf("progress=%#v ok=%v", progress, ok)
	}
}
