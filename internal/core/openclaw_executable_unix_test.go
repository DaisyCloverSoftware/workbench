//go:build !windows

package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindOpenClawExecutableOutsidePATH(t *testing.T) {
	home := t.TempDir()
	bin := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	command := filepath.Join(bin, "openclaw")
	if err := os.WriteFile(command, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("PATH", t.TempDir())

	got, ok := findOpenClawExecutable()
	if !ok {
		t.Fatal("OpenClaw in ~/.local/bin was not discovered when absent from PATH")
	}
	if got != command {
		t.Fatalf("command=%q want %q", got, command)
	}
}

func TestScanProvidersUsesServiceSafeOpenClawDiscovery(t *testing.T) {
	home := t.TempDir()
	bin := filepath.Join(home, ".npm-global", "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	command := filepath.Join(bin, "openclaw")
	if err := os.WriteFile(command, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("PATH", t.TempDir())

	for _, provider := range ScanProviders() {
		if provider.ID != "openclaw" {
			continue
		}
		if !provider.Installed || !provider.Authenticated || provider.Command != command {
			t.Fatalf("unexpected OpenClaw provider: %+v", provider)
		}
		return
	}
	t.Fatal("OpenClaw provider row missing")
}
