package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryPresentationIncludesPriorityInRowsAndDetails(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "item.Priority.String()") || !strings.Contains(text, `"Priority"`) {
		t.Fatal("priority must be visible without relying only on controls")
	}
}
