package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestExecutionTelemetryRefreshUsesExistingEngineSubscription(t *testing.T) {
	body, err := os.ReadFile("production_shell_windows.go")
	if err != nil { t.Fatal(err) }
	text := string(body)
	if !strings.Contains(text, "s.eng.Subscribe") || !strings.Contains(text, "wmAppRefresh") {
		t.Fatal("execution telemetry is not wired through existing engine subscription")
	}
}
