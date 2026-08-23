package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestOperationsClockUsesDistinctAppMessage(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil { t.Fatal(err) }
	if !strings.Contains(string(body), "const wmAppOperationsClock = 0x8002") {
		t.Fatal("operations clock message missing")
	}
}
