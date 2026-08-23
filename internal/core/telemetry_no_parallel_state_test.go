package core

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryPersistsThroughExistingStore(t *testing.T) {
	body, err := os.ReadFile("task_telemetry.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "e.store.Save(st)") {
		t.Fatal("telemetry must use existing task store")
	}
}
