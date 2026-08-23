package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryAcceptanceDoesNotNavigateAwayToObserveProgress(t *testing.T) {
	body, err := os.ReadFile("../../.github/workflows/sprint1-operational-telemetry.yml")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if strings.Count(text, "Click $p 3001") != 1 {
		t.Fatal("acceptance should open Dashboard once and observe live updates in place")
	}
}
