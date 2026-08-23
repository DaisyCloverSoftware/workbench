package core

import (
	"os"
	"strings"
	"testing"
)

func TestStructuredProgressUsesStderrNotFinalStdout(t *testing.T) {
	body, err := os.ReadFile("harness_protocol.go")
	if err != nil { t.Fatal(err) }
	text := string(body)
	if !strings.Contains(text, "cmd.StderrPipe()") || strings.Contains(text, "cmd.StdoutPipe()") {
		t.Fatal("live telemetry must use stderr while final stdout remains result JSON")
	}
}
