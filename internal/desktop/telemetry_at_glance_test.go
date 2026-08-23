package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestCompactTelemetryRowContainsAtGlanceOperationalCondition(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, want := range []string{"operationsTelemetryProgress", "operationsTelemetryElapsed", "operationsActivityAge", "item.Priority.String"} {
		if !strings.Contains(text, want) {
			t.Fatalf("compact row contract missing %q", want)
		}
	}
}
