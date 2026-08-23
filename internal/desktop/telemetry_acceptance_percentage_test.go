package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryAcceptanceRequiresDeterministicPercentInLiveRow(t *testing.T) {
	body, err := os.ReadFile("../../.github/workflows/sprint1-operational-telemetry.yml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "measured percentage in active row") {
		t.Fatal("acceptance does not assert deterministic percentage")
	}
}
