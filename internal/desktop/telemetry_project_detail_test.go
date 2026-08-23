package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestDetailExplicitlyLabelsProject(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil { t.Fatal(err) }
	if !strings.Contains(string(body), `"Project"`) {
		t.Fatal("project detail label missing")
	}
}
