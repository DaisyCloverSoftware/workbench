package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestActivityAgeUsesHumanReadableAgoSuffix(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil { t.Fatal(err) }
	if !strings.Contains(string(body), `return "activity " + telemetryDuration(d) + " ago"`) {
		t.Fatal("human-readable activity age format missing")
	}
}
