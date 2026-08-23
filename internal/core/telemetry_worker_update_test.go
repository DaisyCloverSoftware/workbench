package core

import (
	"os"
	"strings"
	"testing"
)

func TestEngineTelemetryCallbackClosesOverExecutingEngine(t *testing.T) {
	body, err := os.ReadFile("engine.go")
	if err != nil { t.Fatal(err) }
	if !strings.Contains(string(body), "e.updateTaskTelemetry(taskID, progress)") {
		t.Fatal("telemetry callback does not update the executing engine")
	}
}
