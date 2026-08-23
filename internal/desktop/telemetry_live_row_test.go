package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestLiveRowIncludesCurrentStageText(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "phase := strings.TrimSpace(progress.Phase)") {
		t.Fatal("live row does not use current stage/phase")
	}
}
