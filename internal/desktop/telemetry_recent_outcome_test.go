package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryPresentationDoesNotReplaceRecentOutcomes(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil { t.Fatal(err) }
	if strings.Contains(string(body), "RecentOutcomes =") {
		t.Fatal("telemetry presentation must not create a parallel recent-outcome model")
	}
}
