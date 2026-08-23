package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryPresentationUsesCurrentTimeForDerivedAges(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "now := time.Now()") {
		t.Fatal("telemetry presentation does not derive current ages at render time")
	}
}
