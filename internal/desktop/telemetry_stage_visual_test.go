package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestStageTelemetryUsesVisibleFilledAndOpenDots(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, `"●"`) || !strings.Contains(text, `"○"`) {
		t.Fatal("stage progress visual missing")
	}
}
