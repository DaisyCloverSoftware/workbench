package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryDetailRetainsWaitingElapsed(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil { t.Fatal(err) }
	if !strings.Contains(string(body), `"Waiting elapsed"`) {
		t.Fatal("waiting elapsed detail missing")
	}
}
