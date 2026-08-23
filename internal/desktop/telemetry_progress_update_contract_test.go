package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryRefreshPreservesSelectedJobWhileRowsMove(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "if item.ID == operationsDashboardUI.SelectedID") || !strings.Contains(text, "setOperationsList") {
		t.Fatal("telemetry row refresh must preserve selected job")
	}
}
