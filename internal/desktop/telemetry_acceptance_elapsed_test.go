package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryAcceptanceRequiresElapsedRuntimeInLiveRow(t *testing.T) {
	body, err := os.ReadFile("../../.github/workflows/sprint1-operational-telemetry.yml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "elapsed \\d") {
		t.Fatal("acceptance does not assert elapsed runtime")
	}
}
