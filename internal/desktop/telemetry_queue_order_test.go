package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestQueuedTelemetryRowShowsQueueOrder(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `fmt.Sprintf("#%d %s"`) {
		t.Fatal("queue order is not visible in compact row")
	}
}
