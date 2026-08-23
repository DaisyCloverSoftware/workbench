package core

import (
	"context"
	"testing"
)

func TestReportTaskTelemetryDoesNotCallReporterForInvalidProgress(t *testing.T) {
	called := false
	ctx := withTaskTelemetryReporter(context.Background(), func(WorkProgress) { called = true })
	reportTaskTelemetry(ctx, WorkProgress{Kind: ProgressMeasured, Current: 2, Total: 1, Phase: "Invalid"})
	if called {
		t.Fatal("invalid telemetry reached reporter")
	}
}
