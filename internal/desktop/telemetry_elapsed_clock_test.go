package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestElapsedDisplayDerivesFromStartedAtAndCurrentTime(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "now.Sub(*item.StartedAt)") {
		t.Fatal("elapsed display is not derived from task start timestamp")
	}
}
