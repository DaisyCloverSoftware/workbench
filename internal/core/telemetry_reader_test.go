package core

import (
	"os"
	"strings"
	"testing"
)

func TestStructuredHarnessKeepsBoundedOrdinaryStderrCapture(t *testing.T) {
	body, err := os.ReadFile("harness_protocol.go")
	if err != nil { t.Fatal(err) }
	text := string(body)
	if !strings.Contains(text, "newBoundedWorkerCapture(maxWorkerStreamCaptureBytes)") || !strings.Contains(text, `stderr.Write([]byte(line + "\n"))`) {
		t.Fatal("ordinary structured harness stderr capture missing")
	}
}
