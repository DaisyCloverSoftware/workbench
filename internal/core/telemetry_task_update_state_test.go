package core

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryUpdateDoesNotChangeTaskStatus(t *testing.T) {
	body, err := os.ReadFile("task_telemetry.go")
	if err != nil { t.Fatal(err) }
	text := string(body)
	if strings.Contains(text, ".Status =") {
		t.Fatal("telemetry update must not independently change task state")
	}
}
