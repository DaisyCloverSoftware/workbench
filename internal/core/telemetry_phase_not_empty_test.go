package core

import "testing"

func TestWhitespaceOnlyTelemetryPhaseIsRejected(t *testing.T) {
	if progress, ok := normalizeTaskTelemetry(WorkProgress{Kind: ProgressStages, Stage: 1, StageTotal: 2, Phase: "   "}); ok {
		t.Fatalf("blank phase accepted: %#v", progress)
	}
}
