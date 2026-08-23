package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestRunningRowDisplaysNamedPriority(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil { t.Fatal(err) }
	if !strings.Contains(string(body), `priority := strings.ToUpper(item.Priority.String())`) {
		t.Fatal("running row named priority missing")
	}
}
