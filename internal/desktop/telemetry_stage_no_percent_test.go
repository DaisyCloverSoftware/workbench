package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestStagePresentationDoesNotAppendPercentage(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	stage := strings.Index(text, "case core.ProgressStages:")
	if stage < 0 {
		t.Fatal("stage presentation missing")
	}
	segment := text[stage:]
	if end := strings.Index(segment, "return phase"); end >= 0 {
		segment = segment[:end]
	}
	if strings.Contains(segment, "%d%%") {
		t.Fatal("stage branch must not render a percentage")
	}
}
