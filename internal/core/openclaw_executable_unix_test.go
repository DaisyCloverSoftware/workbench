//go:build !windows

package core

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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
		if !strings.Contains(provider.Status, "owner authorization required") {
			t.Fatalf("discovered OpenClaw provider must remain owner-gated: %+v", provider)
		}
		return
	}
	t.Fatal("OpenClaw provider row missing")
}

func TestOpenClawOperationRestoresSiblingNodeToServicePATH(t *testing.T) {
	home := t.TempDir()
	bin := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}

	// Model the common npm/NVM shape: OpenClaw is a script using
	// /usr/bin/env node, but the systemd user service PATH cannot see node.
	node := filepath.Join(bin, "node")
	nodeBody := "#!/bin/sh\necho 'OpenClaw machine operation verified.'\necho 'WORKBENCH_OPERATION_COMPLETE: verified'\n"
	if err := os.WriteFile(node, []byte(nodeBody), 0o755); err != nil {
		t.Fatal(err)
	}
	openclaw := filepath.Join(bin, "openclaw")
	if err := os.WriteFile(openclaw, []byte("#!/usr/bin/env node\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", home)
	t.Setenv("PATH", t.TempDir())
	project := t.TempDir()
	provider := Provider{ID: "openclaw", Name: "OpenClaw", Command: openclaw, Installed: true, Authenticated: true, CanWrite: true, CanRunTools: true}
	task := Task{ID: "task-service-path", Mode: TaskModeOperations, OpenClawOwnerAuthorized: true, ProjectPath: project, Intent: "Verify service execution"}
	sessionID := openClawOperationSessionID(task)

	res, complete, err := runOpenClawOperationInvocation(context.Background(), provider, task, Preferences{}, "verify", sessionID)
	if err != nil {
		t.Fatalf("OpenClaw operation failed with service-like PATH: %v; output=%q", err, res.Output)
	}
	if !complete {
		t.Fatalf("OpenClaw completion marker was not observed: %q", res.Output)
	}
	if !strings.Contains(res.Output, "machine operation verified") {
		t.Fatalf("unexpected operation output: %q", res.Output)
	}
}
