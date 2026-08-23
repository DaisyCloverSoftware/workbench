package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestEngineRefreshImmediatelyRendersTelemetry(t *testing.T) {
	body, err := os.ReadFile("production_shell_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	idx := strings.Index(text, "case wmAppRefresh:")
	if idx < 0 || !strings.Contains(text[idx:], "s.refreshOperationsTelemetryPresentation()") {
		t.Fatal("engine refresh does not render telemetry")
	}
}
