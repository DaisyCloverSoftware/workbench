//go:build windows

package core

import (
	"strings"
	"testing"
)

func TestRunnerSSHConsoleLaunchSpecUsesPersistentWindowsConsole(t *testing.T) {
	spec, err := runnerSSHConsoleLaunchSpec("operator@example.invalid", []string{"$HOME/.local/bin/workbench-runner", "version"})
	if err != nil {
		t.Fatalf("launch spec error: %v", err)
	}
	if spec.File != "cmd.exe" {
		t.Fatalf("launcher file=%q, want cmd.exe", spec.File)
	}
	for _, want := range []string{"/D /K ssh -t", "StrictHostKeyChecking=accept-new", "$HOME/.local/bin/workbench-runner version"} {
		if !strings.Contains(spec.Parameters, want) {
			t.Fatalf("launcher parameters %q missing %q", spec.Parameters, want)
		}
	}
	for _, forbidden := range []string{" start ", " /C ", "BatchMode=yes"} {
		if strings.Contains(" "+spec.Parameters+" ", forbidden) {
			t.Fatalf("persistent console launcher %q unexpectedly contains %q", spec.Parameters, forbidden)
		}
	}
}

func TestRunnerSSHConsoleLaunchSpecRejectsShellMetacharacters(t *testing.T) {
	if _, err := runnerSSHConsoleLaunchSpec("operator@example.invalid", []string{"$HOME/.local/bin/workbench-runner", "version&whoami"}); err == nil {
		t.Fatal("expected unsafe shell token to be rejected")
	}
}
