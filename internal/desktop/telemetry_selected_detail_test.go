package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestSelectedTelemetryDetailUsesSameCanonicalItem(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "surface.Details[operationsDashboardUI.SelectedID]") || !strings.Contains(text, "detail.Item") {
		t.Fatal("selected telemetry detail must come from the same canonical surface item")
	}
}
