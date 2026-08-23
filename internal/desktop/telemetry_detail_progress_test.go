package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestDetailProgressUsesSameMeasuredOrStageRendererAsRows(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(body), "operationsTelemetryProgress(item.Progress, item.State)") < 2 {
		t.Fatal("detail and row must use the same canonical progress rendering")
	}
}
