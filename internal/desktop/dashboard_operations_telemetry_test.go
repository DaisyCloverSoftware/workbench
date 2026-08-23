package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestOperationsTelemetrySourceIncludesLiveClockAndRowMetrics(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, want := range []string{"elapsed ", "activity ", "Stage %d/%d", "%d%%", "queue #%d"} {
		if !strings.Contains(text, want) {
			t.Fatalf("operator row/detail contract missing %q", want)
		}
	}
}
