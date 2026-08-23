package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryClockOnlyRendersOnDashboardPage(t *testing.T) {
	body, err := os.ReadFile("production_shell_windows.go")
	if err != nil { t.Fatal(err) }
	text := string(body)
	idx := strings.Index(text, "case wmAppOperationsClock:")
	if idx < 0 || !strings.Contains(text[idx:], "if s.page == pageDashboard") {
		t.Fatal("operations clock should only render on dashboard")
	}
}
