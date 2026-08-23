package core

import "testing"

func TestMeasuredTelemetryCurrentCannotExceedTotal(t *testing.T) {
	if _, ok := normalizeTaskTelemetry(WorkProgress{Kind: ProgressMeasured, Current: 101, Total: 100, Unit: "files", Phase: "Verifying"}); ok {
		t.Fatal("current greater than total accepted")
	}
}
