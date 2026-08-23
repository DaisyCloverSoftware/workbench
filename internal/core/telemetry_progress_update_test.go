package core

import "testing"

func TestTelemetryNormalizationAllowsSequentialMeasuredUpdates(t *testing.T) {
	for _, current := range []int64{1, 2, 3, 4} {
		progress, ok := normalizeTaskTelemetry(WorkProgress{Kind: ProgressMeasured, Current: current, Total: 4, Unit: "steps", Phase: "Executing steps"})
		if !ok || progress.Current != current {
			t.Fatalf("current=%d progress=%#v ok=%v", current, progress, ok)
		}
	}
}
