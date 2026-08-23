package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestVisibleTelemetrySourceDoesNotContainPercentageUnavailable(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(string(body)), "percentage unavailable") {
		t.Fatal("visible telemetry must use measured or stage progress")
	}
}
