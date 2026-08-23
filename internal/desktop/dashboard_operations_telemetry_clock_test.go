package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestProductionTelemetryClockOnlyRerendersDerivedOperationsTime(t *testing.T) {
	body, err := os.ReadFile("production_shell_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "time.NewTicker(time.Second)") || !strings.Contains(text, "wmAppOperationsClock") {
		t.Fatal("operations live time clock missing")
	}
	if !strings.Contains(text, "Task state changes remain event-driven through Engine.Subscribe") {
		t.Fatal("clock must not replace event-driven execution state updates")
	}
}
