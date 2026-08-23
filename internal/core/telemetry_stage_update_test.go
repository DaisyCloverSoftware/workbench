package core

import "testing"

func TestTelemetryNormalizationAllowsSequentialStageUpdates(t *testing.T) {
	for stage := 1; stage <= 4; stage++ {
		progress, ok := normalizeTaskTelemetry(WorkProgress{Kind: ProgressStages, Stage: stage, StageTotal: 4, Phase: "Executing lifecycle"})
		if !ok || progress.Stage != stage {
			t.Fatalf("stage=%d progress=%#v ok=%v", stage, progress, ok)
		}
	}
}
