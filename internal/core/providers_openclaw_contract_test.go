package core

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestProviderDiscoveryKeepsOpenClawOwnerSelectedAndInertByDefault(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate provider source")
	}
	body, err := os.ReadFile(filepath.Join(filepath.Dir(here), "providers.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, forbidden := range []string{
		"OpenClaw / Harness",
		"configure adapter command",
		"machine-side operations ChatGPT cannot execute itself",
		"falling back to OpenClaw",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("OpenClaw provider discovery contains obsolete automatic-routing copy %q", forbidden)
		}
	}
	for _, want := range []string{
		`Name: "OpenClaw"`,
		"owner-selected machine operations",
		"owner authorization required",
		"inert for automatic routing",
		"durable explicit owner authorization naming OpenClaw",
		"absence does not affect direct Workbench machine controls",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("OpenClaw explicit-owner provider contract missing %q", want)
		}
	}
}
