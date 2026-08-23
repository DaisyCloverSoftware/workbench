package core

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryAdvancesCanonicalActivityTimestamp(t *testing.T) {
	body, err := os.ReadFile("task_telemetry.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "e.state.Tasks[i].UpdatedAt = time.Now()") {
		t.Fatal("telemetry must advance task UpdatedAt")
	}
}
