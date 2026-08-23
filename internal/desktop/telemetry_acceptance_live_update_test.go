package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryAcceptanceComparesSameOpenRowAcrossTime(t *testing.T) {
	body, err := os.ReadFile("../../.github/workflows/sprint1-operational-telemetry.yml")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "$measuredBefore") || !strings.Contains(text, "$measuredAfter") || !strings.Contains(text, "row did not visibly update") {
		t.Fatal("acceptance does not prove live row mutation while dashboard remains open")
	}
}
