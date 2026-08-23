package core

import (
	"context"
	"testing"
)

func TestReportTelemetryWithoutExecutionReporterIsNoOp(t *testing.T) {
	reportTaskTelemetry(context.Background(), WorkProgress{Kind: ProgressMeasured, Current: 1, Total: 2, Phase: "Working"})
}
