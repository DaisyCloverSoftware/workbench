package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestLatestMeaningfulActivityUsesCurrentTelemetryPhase(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "activity := strings.TrimSpace(item.Progress.Phase)") {
		t.Fatal("latest activity does not use current telemetry phase")
	}
}
