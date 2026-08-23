package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryDetailsExposeStartAndElapsed(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, `"Started"`) || !strings.Contains(text, `"Elapsed"`) {
		t.Fatal("start/elapsed detail telemetry missing")
	}
}
