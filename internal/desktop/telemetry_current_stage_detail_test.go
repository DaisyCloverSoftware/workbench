package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestDetailExplicitlyLabelsCurrentStage(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil { t.Fatal(err) }
	if !strings.Contains(string(body), `"Current stage"`) {
		t.Fatal("current stage detail label missing")
	}
}
