package core

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryDocsStateProgressIsValidatedBeforeDurableUpdate(t *testing.T) {
	body, err := os.ReadFile("telemetry_doc.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "validates every record before it can update durable task state") {
		t.Fatal("telemetry validation contract not documented")
	}
}
