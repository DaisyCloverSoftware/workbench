package core

import (
	"context"
	"testing"
)

func TestNilTelemetryReporterContextIsSafe(t *testing.T) {
	ctx := withTaskTelemetryReporter(context.Background(), nil)
	reportTaskTelemetry(ctx, WorkProgress{Kind: ProgressStages, Stage: 1, StageTotal: 2, Phase: "Working"})
}
