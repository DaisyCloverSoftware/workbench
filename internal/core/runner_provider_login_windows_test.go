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
	for _, want := range []string{"cmd.exe /D /K ssh -t", "StrictHostKeyChecking=accept-new", "$HOME/.local/bin/workbench-runner version"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("launcher args %q missing %q", joined, want)
		}
	}
	for _, forbidden := range []string{" start ", " /C ", "BatchMode=yes"} {
		if strings.Contains(" "+joined+" ", forbidden) {
			t.Fatalf("persistent console launcher %q unexpectedly contains %q", joined, forbidden)
		}
	}
	if cmd.SysProcAttr == nil {
		t.Fatal("persistent console launcher must configure Windows process attributes")
	}
	if cmd.SysProcAttr.CreationFlags&0x00000010 == 0 { // CREATE_NEW_CONSOLE
		t.Fatalf("persistent console launcher must request CREATE_NEW_CONSOLE, flags=%#x", cmd.SysProcAttr.CreationFlags)
	}
	if cmd.SysProcAttr.CreationFlags&0x08000000 != 0 { // CREATE_NO_WINDOW
		t.Fatalf("persistent console launcher must not request CREATE_NO_WINDOW, flags=%#x", cmd.SysProcAttr.CreationFlags)
	}
	if cmd.SysProcAttr.HideWindow {
		t.Fatal("persistent console launcher must not hide the interactive console")
	}
}

func TestRunnerSSHConsoleLauncherRejectsShellMetacharacters(t *testing.T) {
	if _, err := runnerSSHConsoleLauncher("operator@example.invalid", []string{"$HOME/.local/bin/workbench-runner", "version&whoami"}); err == nil {
		t.Fatal("expected unsafe shell token to be rejected")
	}
}
