package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryAcceptanceRequiresMeasuredAndStageProgress(t *testing.T) {
	body, err := os.ReadFile("../../.github/workflows/sprint1-operational-telemetry.yml")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, want := range []string{"Telemetry measured", "Telemetry stage", "measured percentage in active row", "stage progress in operation row", "visible priority change"} {
		if !strings.Contains(text, want) {
			t.Fatalf("telemetry acceptance missing %q", want)
		}
	}
}
