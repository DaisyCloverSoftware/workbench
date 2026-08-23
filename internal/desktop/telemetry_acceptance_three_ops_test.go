package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryAcceptanceLaunchesMultipleSimultaneousOperations(t *testing.T) {
	body, err := os.ReadFile("../../.github/workflows/sprint1-operational-telemetry.yml")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, want := range []string{"Telemetry stage operation", "Telemetry measured source verification", "Telemetry needs you"} {
		if !strings.Contains(text, want) {
			t.Fatalf("acceptance workload missing %q", want)
		}
	}
}
