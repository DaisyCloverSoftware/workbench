package core

import (
	"os"
	"strings"
	"testing"
)

func TestHarnessTelemetryScannerHasExplicitBound(t *testing.T) {
	body, err := os.ReadFile("harness_protocol.go")
	if err != nil { t.Fatal(err) }
	if !strings.Contains(string(body), "scanner.Buffer(make([]byte, 4096), maxWorkerStreamCaptureBytes)") {
		t.Fatal("telemetry scanner bound missing")
	}
}
