package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestMeasuredPercentDerivesFromCurrentOverTotal(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "percent := int((current * 100) / progress.Total)") {
		t.Fatal("measured percent is not derived from real current/total telemetry")
	}
}
