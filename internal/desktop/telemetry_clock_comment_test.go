package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestOperationsClockCommentDistinguishesStateEventsFromTimeRerender(t *testing.T) {
	body, err := os.ReadFile("production_shell_windows.go")
	if err != nil { t.Fatal(err) }
	text := string(body)
	if !strings.Contains(text, "only re-renders time-derived elapsed/activity ages") {
		t.Fatal("operations clock architecture explanation missing")
	}
}
