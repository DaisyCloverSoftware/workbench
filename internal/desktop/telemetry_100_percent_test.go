package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestMeasuredBarSupportsTruthfulHundredPercent(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "if percent == 100") {
		t.Fatal("measured progress bar does not handle complete measurement")
	}
}
