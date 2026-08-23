package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestSelectedDetailShowsPriorityForRunningAndQueuedTasks(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "priority := item.Priority.String()") || !strings.Contains(text, `writeOperationsDetailLine(&b, "Priority", priority)`) {
		t.Fatal("selected detail priority missing")
	}
}
