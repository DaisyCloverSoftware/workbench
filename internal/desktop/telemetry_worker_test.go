package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestExpandedTelemetryRowIncludesAssignedWorker(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `" · worker "`) {
		t.Fatal("expanded telemetry row must expose assigned worker")
	}
}
