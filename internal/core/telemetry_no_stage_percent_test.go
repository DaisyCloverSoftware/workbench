package core

import "testing"

func TestStageTelemetryClearsMeasuredFields(t *testing.T) {
	progress, ok := normalizeTaskTelemetry(WorkProgress{Kind: ProgressStages, Current: 9, Total: 10, Unit: "files", Stage: 2, StageTotal: 4, Phase: "Executing"})
	if !ok {
		t.Fatal("valid stage telemetry rejected")
	}
	if progress.Current != 0 || progress.Total != 0 || progress.Unit != "" {
		t.Fatalf("stage telemetry retained measured fields: %#v", progress)
	}
}
