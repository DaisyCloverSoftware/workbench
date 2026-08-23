package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestActiveRowProgressDoesNotDependOnSelectedID(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	idx := strings.Index(text, "func operationsTelemetryListLine")
	if idx < 0 {
		t.Fatal("active row formatter missing")
	}
	segment := text[idx:]
	if end := strings.Index(segment, "func operationsTelemetryExpandedLine"); end >= 0 { segment = segment[:end] }
	if strings.Contains(segment, "SelectedID") {
		t.Fatal("active row progress must not require task selection")
	}
}
