package core

import (
	"context"
	"testing"
)

func TestReportTaskTelemetryCallsReporterForValidProgress(t *testing.T) {
	called := false
	ctx := withTaskTelemetryReporter(context.Background(), func(progress WorkProgress) { called = progress.Stage == 2 })
	reportTaskTelemetry(ctx, WorkProgress{Kind: ProgressStages, Stage: 2, StageTotal: 4, Phase: "Executing"})
	if !called {
		t.Fatal("valid telemetry did not reach reporter")
	}
}
