package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestDetailExplicitlyLabelsElapsedRuntime(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil { t.Fatal(err) }
	if !strings.Contains(string(body), `"Elapsed"`) {
		t.Fatal("elapsed detail label missing")
	}
}
