package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryPresentationDoesNotCreateSeparateProjectActivityState(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil { t.Fatal(err) }
	if strings.Contains(string(body), "Projects =") {
		t.Fatal("telemetry presentation must not maintain parallel project activity state")
	}
}
