package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestOperationsClockDoesNotRebuildExecutionState(t *testing.T) {
	body, err := os.ReadFile("production_shell_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	needle := "case wmAppOperationsClock:"
	idx := strings.Index(text, needle)
	if idx < 0 {
		t.Fatal("operations clock case missing")
	}
	segment := text[idx:]
	if end := strings.Index(segment, "case wmCommand:"); end >= 0 {
		segment = segment[:end]
	}
	if strings.Contains(segment, "s.refresh()") || strings.Contains(segment, "BuildDashboardOperationsSurface") {
		t.Fatal("clock must only re-render derived elapsed/activity age")
	}
}
