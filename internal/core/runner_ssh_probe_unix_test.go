//go:build !windows

package core

import (
	"os"
	"strings"
	"testing"
)

func TestWorkbenchRunnerSSHUsesFixedRunnerVersionCommand(t *testing.T) {
	_, logPath := installFakeRunnerSSH(t, `
printf '%s\n' "$*" >> "$FAKE_SSH_LOG"
case "$*" in
  *"$HOME/.local/bin/workbench-runner version") printf '%s\n' '0.8.0' ;;
  *) exit 2 ;;
esac`)

	version, err := TestWorkbenchRunnerSSH("runner.example")
	if err != nil {
		t.Fatal(err)
	}
	if version != "0.8.0" {
		t.Fatalf("runner version=%q want 0.8.0", version)
	}
	calls, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(calls)
	if !strings.Contains(text, "$HOME/.local/bin/workbench-runner version") {
		t.Fatalf("runner version command missing: %s", text)
	}
	if strings.Contains(strings.ToLower(text), "openclaw") {
		t.Fatalf("runner probe invoked OpenClaw instead of Workbench Runner: %s", text)
	}
}
