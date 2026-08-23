package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryPresentationReadsExistingOperationsSurface(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil { t.Fatal(err) }
	if !strings.Contains(string(body), "surface := operationsDashboardUI.Surface") {
		t.Fatal("telemetry presentation must reuse existing Operations surface")
	}
}
