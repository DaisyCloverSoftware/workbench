package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestDetailExplicitlyLabelsTask(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil { t.Fatal(err) }
	if !strings.Contains(string(body), `"Task / sprint"`) {
		t.Fatal("task detail label missing")
	}
}
