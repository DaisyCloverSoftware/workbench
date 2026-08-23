package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryDetailsExposeTimestampAndHumanActivityAge(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, `Format("2 Jan 15:04:05")`) || !strings.Contains(text, "operationsActivityAge") {
		t.Fatal("latest activity timestamp/age missing")
	}
}
