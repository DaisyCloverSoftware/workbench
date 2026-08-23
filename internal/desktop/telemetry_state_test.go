package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryRowsRetainCurrentState(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "strings.ToUpper(dashboardStatusLabel") {
		t.Fatal("current state missing from telemetry row")
	}
}
