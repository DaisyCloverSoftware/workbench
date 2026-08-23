package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryRowKeepsProjectAndTaskIdentity(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "operationsProjectLabel(item)") || !strings.Contains(text, "item.Title") {
		t.Fatal("project/task identity missing from active row")
	}
}
