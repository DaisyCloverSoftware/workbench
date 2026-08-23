package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestMeasuredBarFillUsesDerivedPercent(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil { t.Fatal(err) }
	if !strings.Contains(string(body), "filled := percent * width / 100") {
		t.Fatal("measured bar fill is not derived from percent")
	}
}
