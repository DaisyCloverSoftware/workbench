package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryDetailsRetainCIReference(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "CI / build / deployment reference") {
		t.Fatal("CI/build/deployment reference missing from telemetry details")
	}
}
