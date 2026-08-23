package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryDetailsRetainWaitingReasonAndAutomaticCheck(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, `"Waiting for"`) || !strings.Contains(text, `"Automatic retry / check"`) {
		t.Fatal("waiting telemetry context missing")
	}
}
