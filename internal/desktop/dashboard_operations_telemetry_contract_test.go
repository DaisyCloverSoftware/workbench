package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestSprint1OperationsTelemetryPresentationContract(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, want := range []string{"ProgressMeasured", "telemetryBar", "ProgressStages", "telemetryStageDots", "operationsActivityAge", "operationsTelemetryElapsed", "Priority"} {
		if !strings.Contains(text, want) {
			t.Fatalf("telemetry presentation missing %q", want)
		}
	}
	if strings.Contains(strings.ToLower(text), "deterministic percentage unavailable") {
		t.Fatal("telemetry presentation must not expose percentage-unavailable copy")
	}
}
