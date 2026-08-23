package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestDetailExplicitlyLabelsProgress(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil { t.Fatal(err) }
	if !strings.Contains(string(body), `"Progress"`) {
		t.Fatal("progress detail label missing")
	}
}
