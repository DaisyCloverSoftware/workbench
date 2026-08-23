package core

import (
	"os"
	"strings"
	"testing"
)

func TestHarnessProgressReporterIsBoundToExecutingTaskID(t *testing.T) {
	body, err := os.ReadFile("engine.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "e.updateTaskTelemetry(taskID, progress)") {
		t.Fatal("progress reporter is not bound to executing task id")
	}
}
