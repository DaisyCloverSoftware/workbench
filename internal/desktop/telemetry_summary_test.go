package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryIsNotHiddenOnlyInDetailPane(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	idx := strings.Index(text, "func operationsTelemetryListLine")
	if idx < 0 {
		t.Fatal("compact telemetry row formatter missing")
	}
	segment := text[idx:]
	if end := strings.Index(segment, "func operationsTelemetryExpandedLine"); end >= 0 {
		segment = segment[:end]
	}
	for _, want := range []string{"operationsTelemetryProgress", "operationsTelemetryElapsed", "operationsActivityAge", "priority"} {
		if !strings.Contains(segment, want) {
			t.Fatalf("compact row missing %q", want)
		}
	}
}
