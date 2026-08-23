package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryAcceptanceCrossChecksDurableTaskState(t *testing.T) {
	body, err := os.ReadFile("../../.github/workflows/sprint1-operational-telemetry.yml")
	if err != nil { t.Fatal(err) }
	text := string(body)
	if !strings.Contains(text, "function State()") || !strings.Contains(text, ".tasks") {
		t.Fatal("telemetry acceptance does not cross-check durable task state")
	}
}
