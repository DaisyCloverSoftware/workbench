package core

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryUpdatesExistingTaskProgressAndUpdatedAt(t *testing.T) {
	body, err := os.ReadFile("task_telemetry.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, ".Progress = progress") || !strings.Contains(text, ".UpdatedAt = time.Now()") {
		t.Fatal("telemetry must persist into canonical task fields")
	}
}
