package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryAcceptanceMeasuredEvidenceMentionsProgressBar(t *testing.T) {
	body, err := os.ReadFile("../../.github/workflows/sprint1-operational-telemetry.yml")
	if err != nil { t.Fatal(err) }
	if !strings.Contains(string(body), "percentage/progress bar") {
		t.Fatal("acceptance evidence does not identify measured progress bar")
	}
}
