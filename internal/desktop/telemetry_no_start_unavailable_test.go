package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestNewTelemetryPresentationDoesNotUseStartTimeUnavailableCopy(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil { t.Fatal(err) }
	if strings.Contains(strings.ToLower(string(body)), "start time unavailable") {
		t.Fatal("new telemetry presentation must show real start data when available without legacy unavailable copy")
	}
}
