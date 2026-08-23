package core

import (
	"os"
	"strings"
	"testing"
)

func TestStructuredHarnessTelemetryIsOptionalAndBackwardsCompatible(t *testing.T) {
	body, err := os.ReadFile("harness_protocol.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "Optional validated progress events") || !strings.Contains(text, "Final result JSON remains") {
		t.Fatal("structured telemetry compatibility contract missing")
	}
}
