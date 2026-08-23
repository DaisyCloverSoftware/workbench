package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestPriorityCommandImmediatelyRendersUpdatedTelemetryRow(t *testing.T) {
	body, err := os.ReadFile("production_shell_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	idx := strings.Index(text, "if s.handleOperationsDashboardCommand(id, notify)")
	if idx < 0 || !strings.Contains(text[idx:], "s.refreshOperationsTelemetryPresentation()") {
		t.Fatal("operations action does not immediately rerender telemetry")
	}
}
