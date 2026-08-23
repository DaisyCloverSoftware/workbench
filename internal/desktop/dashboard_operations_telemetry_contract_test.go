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
	for _, want := range []string{"ProgressMeasured", "telemetryBar", "ProgressStages", "telemetryStageDots", "operationsActivityAge", "operationsTelemetryElapsed", "Priority", "redrawOperationsTelemetryControls", "rdwUpdateNow"} {
		if !strings.Contains(text, want) {
			t.Fatalf("telemetry presentation missing %q", want)
		}
	}
	if strings.Contains(strings.ToLower(text), "deterministic percentage unavailable") {
		t.Fatal("telemetry presentation must not expose percentage-unavailable copy")
	}
}

func TestSprint1OperationsCanonicalLayoutKeepsCompactLanesWide(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_controls_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, want := range []string{
		"colW := (contentW - colGap) / 2",
		"rowH := (boardH - rowGap*2) / 3",
		"col := i % 2",
		"row := i / 2",
		"operationsTelemetryListLine(item, time.Now())",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("canonical Operations layout missing %q", want)
		}
	}
	if strings.Contains(text, "colW := (contentW - colGap*2) / 3") {
		t.Fatal("canonical Operations layout regressed to three narrow columns")
	}
}
