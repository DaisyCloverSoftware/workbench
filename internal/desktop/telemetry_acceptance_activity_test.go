package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryAcceptanceRequiresActivityAgeInLiveRow(t *testing.T) {
	body, err := os.ReadFile("../../.github/workflows/sprint1-operational-telemetry.yml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "activity \\d") {
		t.Fatal("acceptance does not assert live activity age")
	}
}
