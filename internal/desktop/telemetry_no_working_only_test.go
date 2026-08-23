package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestRunningTelemetryRowContainsProgressBeyondWorkingState(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil { t.Fatal(err) }
	text := string(body)
	if !strings.Contains(text, "operationsTelemetryProgress") || !strings.Contains(text, "Stage %d/%d") {
		t.Fatal("running telemetry row lacks progress beyond state label")
	}
}
