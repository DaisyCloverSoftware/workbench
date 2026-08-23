package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestMeasuredProgressUsesVisibleTwelveCellBar(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil { t.Fatal(err) }
	if !strings.Contains(string(body), "telemetryBar(percent, 12)") {
		t.Fatal("measured row progress bar width missing")
	}
}
