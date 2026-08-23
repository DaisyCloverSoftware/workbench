package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryRowUsesCanonicalWorkItemState(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil { t.Fatal(err) }
	if !strings.Contains(string(body), "dashboardStatusLabel(item.State") {
		t.Fatal("telemetry row does not use canonical work item state")
	}
}
