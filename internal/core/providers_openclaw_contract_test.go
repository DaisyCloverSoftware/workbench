package core

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestProviderDiscoveryKeepsOpenClawOperationsOnly(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate provider source")
	}
	body, err := os.ReadFile(filepath.Join(filepath.Dir(here), "providers.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, forbidden := range []string{"OpenClaw / Harness", "configure adapter command", "coding agent / reviewer"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("OpenClaw provider discovery contains obsolete coding/harness copy %q", forbidden)
		}
	}
	for _, want := range []string{`Name: "OpenClaw"`, `Status: "not detected"`, "machine-side operations harness", "machine-side operations ChatGPT cannot execute itself"} {
		if !strings.Contains(text, want) {
			t.Fatalf("OpenClaw operations discovery contract missing %q", want)
		}
	}
}
