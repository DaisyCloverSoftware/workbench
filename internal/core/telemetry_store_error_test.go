package core

import "testing"

func TestUnsupportedTelemetryIsRejectedBeforeStoreWrite(t *testing.T) {
	if _, ok := normalizeTaskTelemetry(WorkProgress{Kind: ProgressKind("unknown"), Phase: "Unknown"}); ok {
		t.Fatal("unsupported telemetry accepted")
	}
}
