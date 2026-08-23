package core

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryUpdateNotifiesExistingEngineSubscribers(t *testing.T) {
	body, err := os.ReadFile("task_telemetry.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "e.notify()") {
		t.Fatal("telemetry must use existing engine notification architecture")
	}
}
