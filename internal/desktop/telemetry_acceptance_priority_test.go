package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryAcceptanceProvesImmediatePriorityChange(t *testing.T) {
	body, err := os.ReadFile("../../.github/workflows/sprint1-operational-telemetry.yml")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "Click $p 3615") || !strings.Contains(text, "visible priority change") {
		t.Fatal("acceptance does not exercise Priority Up and visible row update")
	}
}
