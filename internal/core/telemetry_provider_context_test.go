package core

import (
	"os"
	"strings"
	"testing"
)

func TestEngineWrapsProviderRunWithTelemetryReporter(t *testing.T) {
	body, err := os.ReadFile("engine.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "withTaskTelemetryReporter(runCtx") || !strings.Contains(text, "e.updateTaskTelemetry(taskID, progress)") {
		t.Fatal("provider execution telemetry reporter is not wired to canonical engine state")
	}
}
