package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryCorrectionIsNotDashboardRedesign(t *testing.T) {
	body, err := os.ReadFile("../../SPRINT1_TELEMETRY_NOTES.md")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(string(body)), "redesign") {
		t.Fatal("telemetry correction should remain focused on execution telemetry")
	}
}
