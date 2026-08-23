package core

import (
	"context"
	"testing"
)

func TestTelemetryReporterContextPreservesCancellation(t *testing.T) {
	base, cancel := context.WithCancel(context.Background())
	ctx := withTaskTelemetryReporter(base, func(WorkProgress) {})
	cancel()
	if ctx.Err() != context.Canceled {
		t.Fatalf("context cancellation lost: %v", ctx.Err())
	}
}
