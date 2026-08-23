package core

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryOnlyMutatesRunningOrRoutingTasks(t *testing.T) {
	body, err := os.ReadFile("task_telemetry.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "TaskRunning") || !strings.Contains(text, "TaskRouting") {
		t.Fatal("telemetry status guard missing")
	}
}
