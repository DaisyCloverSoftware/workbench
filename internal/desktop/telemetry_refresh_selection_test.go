package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryFullLaneRefreshPreservesSelectedID(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil { t.Fatal(err) }
	text := string(body)
	if strings.Count(text, "item.ID == operationsDashboardUI.SelectedID") < 2 {
		t.Fatal("compact and full lane refresh should both preserve selected id")
	}
}
