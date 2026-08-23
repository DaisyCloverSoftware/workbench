package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryCorrectionKeepsExistingPriorityControls(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_controls_windows.go")
	if err != nil { t.Fatal(err) }
	text := string(body)
	if !strings.Contains(text, "idOpsPriorityUp") || !strings.Contains(text, "SetTaskPriority") {
		t.Fatal("priority controls no longer connect to scheduler state")
	}
}
