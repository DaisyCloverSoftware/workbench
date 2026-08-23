package core

import "testing"

func TestNormalHarnessStderrIsNotParsedAsTelemetry(t *testing.T) {
	if progress, ok := parseHarnessProgressLine("ordinary worker diagnostic"); ok {
		t.Fatalf("ordinary stderr parsed as telemetry: %#v", progress)
	}
}
