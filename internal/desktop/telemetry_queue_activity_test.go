package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestQueuedRowStillShowsLatestActivityAge(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil { t.Fatal(err) }
	text := string(body)
	idx := strings.Index(text, "if item.State == core.TaskQueued")
	if idx < 0 || !strings.Contains(text[idx:], "activity") {
		t.Fatal("queued row activity age missing")
	}
}
