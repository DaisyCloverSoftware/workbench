package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryRowsExposeComparableMeasuredAndStagePosition(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "%d%%") || !strings.Contains(text, "Stage %d/%d") {
		t.Fatal("rows must expose comparable progress position")
	}
}
