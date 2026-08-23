package core

import (
	"context"
	"testing"
)

func TestTelemetryReporterFlowsThroughExecutionContext(t *testing.T) {
	var got WorkProgress
	ctx := withTaskTelemetryReporter(context.Background(), func(progress WorkProgress) { got = progress })
	reportTaskTelemetry(ctx, WorkProgress{Kind: ProgressStages, Stage: 2, StageTotal: 4, Phase: "Executing"})
	if got.Stage != 2 || got.StageTotal != 4 || got.Phase != "Executing" {
		t.Fatalf("got=%#v", got)
	}
}
