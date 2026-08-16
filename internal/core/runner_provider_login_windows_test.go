//go:build windows

package core

import (
	"strings"
	"testing"
)

func TestRunnerSSHConsoleLauncherUsesPersistentWindowsConsole(t *testing.T) {
	cmd, err := runnerSSHConsoleLauncher("operator@example.invalid", []string{"$HOME/.local/bin/workbench-runner", "version"})
	if err != nil {
		t.Fatalf("launcher error: %v", err)
	}
	joined := strings.Join(cmd.Args, " ")
	for _, want := range []string{"cmd.exe", `start "" cmd.exe /D /K`, "ssh -t", "StrictHostKeyChecking=accept-new", "$HOME/.local/bin/workbench-runner version"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("launcher args %q missing %q", joined, want)
		}
	}
	if strings.Contains(joined, `\"`) {
		t.Fatalf("cmd.exe launcher must not use backslash-escaped quotes: %q", joined)
	}
	if strings.Contains(joined, "BatchMode=yes") {
		t.Fatalf("interactive launcher must not force BatchMode: %q", joined)
	}
}

func TestRunnerSSHConsoleLauncherRejectsShellMetacharacters(t *testing.T) {
	if _, err := runnerSSHConsoleLauncher("operator@example.invalid", []string{"$HOME/.local/bin/workbench-runner", "version&whoami"}); err == nil {
		t.Fatal("expected unsafe shell token to be rejected")
	}
}
