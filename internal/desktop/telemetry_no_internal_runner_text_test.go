package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryRowDoesNotRenderAttemptsOrRawRunnerOutput(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	idx := strings.Index(text, "func operationsTelemetryListLine")
	if idx < 0 { t.Fatal("row formatter missing") }
	segment := text[idx:]
	if end := strings.Index(segment, "func operationsTelemetryExpandedLine"); end >= 0 { segment = segment[:end] }
	if strings.Contains(segment, "Attempts") || strings.Contains(segment, "Output") {
		t.Fatal("compact telemetry row must not require interpreting raw runner text")
	}
}
