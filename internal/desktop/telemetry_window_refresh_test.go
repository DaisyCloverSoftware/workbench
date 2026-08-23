package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestProductionShellRedrawsAfterTelemetryRefresh(t *testing.T) {
	body, err := os.ReadFile("production_shell_windows.go")
	if err != nil { t.Fatal(err) }
	text := string(body)
	idx := strings.Index(text, "case wmAppRefresh:")
	if idx < 0 { t.Fatal("refresh case missing") }
	segment := text[idx:]
	if !strings.Contains(segment, "s.refreshOperationsTelemetryPresentation()") || !strings.Contains(segment, "redrawProductionWindow(hwnd)") {
		t.Fatal("engine refresh does not render/redraw telemetry")
	}
}
