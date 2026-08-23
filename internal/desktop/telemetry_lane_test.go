package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryPresentationRewritesEveryOperationsLane(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "for _, control := range operationsLaneControls") {
		t.Fatal("telemetry presentation must apply to every Operations lane")
	}
}
