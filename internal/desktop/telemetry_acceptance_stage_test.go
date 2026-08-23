package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryAcceptanceRequiresStageProgressInLiveRow(t *testing.T) {
	body, err := os.ReadFile("../../.github/workflows/sprint1-operational-telemetry.yml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "Stage \\d+/4") {
		t.Fatal("acceptance does not assert stage progress")
	}
}
