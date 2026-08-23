package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryDetailsRetainOwnerAction(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"Owner action"`) {
		t.Fatal("owner action missing from telemetry details")
	}
}
