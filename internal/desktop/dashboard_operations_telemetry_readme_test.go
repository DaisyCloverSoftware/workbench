package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestSprint1TelemetryCorrectionNotesKeepCanonicalState(t *testing.T) {
	body, err := os.ReadFile("../../SPRINT1_TELEMETRY_NOTES.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "does not introduce a parallel dashboard state model") {
		t.Fatal("telemetry correction must stay on canonical task state")
	}
}
