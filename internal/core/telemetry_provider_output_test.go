package core

import (
	"os"
	"strings"
	"testing"
)

func TestHarnessFinalReportRemainsSeparateFromLiveProgress(t *testing.T) {
	body, err := os.ReadFile("harness_protocol.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "Report      string") || !strings.Contains(text, "HarnessProgress") {
		t.Fatal("final report/live progress contracts missing")
	}
}
