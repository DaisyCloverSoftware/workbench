package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestProductionShellRendersTelemetryImmediatelyOnWindowCreate(t *testing.T) {
	body, err := os.ReadFile("production_shell_windows.go")
	if err != nil { t.Fatal(err) }
	text := string(body)
	idx := strings.Index(text, "case wmCreate:")
	if idx < 0 || !strings.Contains(text[idx:], "s.refreshOperationsTelemetryPresentation()") {
		t.Fatal("telemetry is not rendered on dashboard creation")
	}
}
