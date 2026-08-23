package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryDetailsRetainFailureInformation(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"Failure"`) {
		t.Fatal("failure information missing from telemetry details")
	}
}
