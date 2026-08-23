package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestExpandedTelemetryUsesWorkItemProvider(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "worker := strings.TrimSpace(item.Provider)") {
		t.Fatal("worker provider missing from expanded telemetry")
	}
}
