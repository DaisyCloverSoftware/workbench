package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryPresentationDoesNotCreateSeparateWorkerAssignmentState(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil { t.Fatal(err) }
	if strings.Contains(string(body), "Workers =") {
		t.Fatal("telemetry presentation must not maintain parallel worker assignment state")
	}
}
