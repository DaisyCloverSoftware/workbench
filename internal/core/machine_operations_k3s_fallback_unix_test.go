package core

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInspectMachineRetriesValidatedKubectlThroughSanctionedK3sSudo(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX fake executable smoke")
	}
	bin := t.TempDir()
	kubectl := filepath.Join(bin, "kubectl")
	sudo := filepath.Join(bin, "sudo")
	if err := os.WriteFile(kubectl, []byte("#!/bin/sh\necho 'Unable to read /etc/rancher/k3s/k3s.yaml: permission denied' >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sudo, []byte("#!/bin/sh\nprintf 'sudo-argv=%s\\n' \"$*\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	result, err := InspectMachine(context.Background(), MachineCommandRequest{Program: "kubectl", Args: []string{"get", "nodes", "-o", "wide"}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Transport != "k3s-sudo" {
		t.Fatalf("transport=%q want k3s-sudo", result.Transport)
	}
	if !strings.Contains(result.Output, "sudo-argv=-n k3s kubectl get nodes -o wide") {
		t.Fatalf("unexpected elevated argv: %q", result.Output)
	}
}

func TestInspectMachineDoesNotElevateGenericKubectlFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX fake executable smoke")
	}
	bin := t.TempDir()
	kubectl := filepath.Join(bin, "kubectl")
	sudo := filepath.Join(bin, "sudo")
	marker := filepath.Join(bin, "sudo-called")
	if err := os.WriteFile(kubectl, []byte("#!/bin/sh\necho 'connection refused' >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sudo, []byte("#!/bin/sh\ntouch '"+marker+"'\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	result, err := InspectMachine(context.Background(), MachineCommandRequest{Program: "kubectl", Args: []string{"get", "nodes"}})
	if err == nil {
		t.Fatal("generic kubectl failure should remain an error")
	}
	if result.Transport != "direct" {
		t.Fatalf("transport=%q want direct", result.Transport)
	}
	if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
		t.Fatal("sudo must not run for a generic kubectl failure")
	}
}

func TestMachineCommandPolicyNeverExposesSudoProgram(t *testing.T) {
	if _, err := validateMachineCommand(MachineCommandRequest{Program: "sudo", Args: []string{"-n", "k3s", "kubectl", "get", "nodes"}}); err == nil {
		t.Fatal("sudo must never be directly callable through Workbench machine commands")
	}
}
