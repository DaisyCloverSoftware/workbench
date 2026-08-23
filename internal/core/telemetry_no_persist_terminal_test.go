package core

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryMutationGuardMentionsOnlyActiveStatuses(t *testing.T) {
	body, err := os.ReadFile("task_telemetry.go")
	if err != nil { t.Fatal(err) }
	text := string(body)
	guard := strings.Index(text, "if i < 0 ||")
	if guard < 0 { t.Fatal("telemetry active-status guard missing") }
	segment := text[guard:]
	if !strings.Contains(segment, "TaskRunning") || !strings.Contains(segment, "TaskRouting") {
		t.Fatal("active-status guard incomplete")
	}
}
