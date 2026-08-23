package core

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryCanonicalStoreSaveRemainsVisibleInImplementation(t *testing.T) {
	body, err := os.ReadFile("task_telemetry.go")
	if err != nil { t.Fatal(err) }
	if !strings.Contains(string(body), "_ = e.store.Save(st)") {
		t.Fatal("canonical store save missing")
	}
}
