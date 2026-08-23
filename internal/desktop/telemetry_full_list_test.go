package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestFullLaneUsesSameTelemetryLineAsCompactLanes(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil { t.Fatal(err) }
	if !strings.Contains(string(body), "operationsTelemetryExpandedLine(item, now)") {
		t.Fatal("full lane does not use telemetry-aware line formatting")
	}
}
