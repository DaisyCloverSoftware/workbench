package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestDetailExplicitlyLabelsAssignedWorker(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil { t.Fatal(err) }
	if !strings.Contains(string(body), `"Assigned worker"`) {
		t.Fatal("assigned worker detail label missing")
	}
}
