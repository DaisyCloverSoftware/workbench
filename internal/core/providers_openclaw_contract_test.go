package core

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestProviderDiscoveryDoesNotRestoreLegacyOpenClawHarnessCopy(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate provider source")
	}
	body, err := os.ReadFile(filepath.Join(filepath.Dir(here), "providers.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, forbidden := range []string{"OpenClaw / Harness", "configure adapter command"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("provider discovery restored obsolete combined harness copy %q", forbidden)
		}
	}
	for _, want := range []string{`Name: "OpenClaw"`, `Status: "not detected"`, "Structured harness adapters are configured separately."} {
		if !strings.Contains(text, want) {
			t.Fatalf("OpenClaw discovery contract missing %q", want)
		}
	}
}
