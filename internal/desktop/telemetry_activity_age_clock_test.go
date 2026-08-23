package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestActivityAgeRerendersOnOperationsClock(t *testing.T) {
	body, err := os.ReadFile("production_shell_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	idx := strings.Index(text, "case wmAppOperationsClock:")
	if idx < 0 || !strings.Contains(text[idx:], "s.refreshOperationsTelemetryPresentation()") {
		t.Fatal("activity age is not rerendered by the Operations clock")
	}
}
