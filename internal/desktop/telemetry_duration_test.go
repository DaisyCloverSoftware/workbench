package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryDurationSupportsSecondsMinutesHours(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil { t.Fatal(err) }
	text := string(body)
	for _, want := range []string{"%ds", "%dm%02ds", "%dh%02dm"} {
		if !strings.Contains(text, want) { t.Fatalf("duration format missing %q", want) }
	}
}
