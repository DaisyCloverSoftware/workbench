//go:build !windows

package core

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenClawOperationsSupervisorReusesAndArchivesOneJobConversation(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "fake-openclaw")
	body := `#!/usr/bin/env bash
set -euo pipefail
script_dir="$(cd "$(dirname "$0")" && pwd)"
if [ "${1:-}" = "sessions" ] && [ "${2:-}" = "archive" ]; then
  printf '%s' "${3:-}" > "$script_dir/.workbench-fake-archive"
  exit 0
fi
count_file="$PWD/.workbench-fake-count"
session_file="$PWD/.workbench-fake-session"
count=0
if [ -f "$count_file" ]; then count="$(cat "$count_file")"; fi
count=$((count+1))
printf '%s' "$count" > "$count_file"
session=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--session-id" ] && [ "$#" -gt 1 ]; then session="$2"; shift 2; continue; fi
  shift
done
if [ -z "$session" ]; then echo 'missing session id' >&2; exit 2; fi
if [ -f "$session_file" ] && [ "$(cat "$session_file")" != "$session" ]; then echo 'session changed during one job' >&2; exit 3; fi
printf '%s' "$session" > "$session_file"
if [ "$count" -eq 1 ]; then
  echo "Restart complete; still checking health."
  exit 0
fi
echo "Restart and health verification complete."
echo "WORKBENCH_OPERATION_COMPLETE: verified"
`
	if err := os.WriteFile(script, []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}

	provider := Provider{ID: "openclaw", Name: "OpenClaw", Command: script, Installed: true, Authenticated: true, CanWrite: true, CanRunTools: true}
	task := Task{ID: "task-supervised-op", Mode: TaskModeOperations, ProjectPath: dir, Intent: "Restart the runner and verify health"}
	res, err := RunOpenClawOperationSupervised(context.Background(), provider, task, Preferences{})
	if err != nil {
		t.Fatalf("supervised operation failed: %v; output=%q", err, res.Output)
	}
	count, err := os.ReadFile(filepath.Join(dir, ".workbench-fake-count"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(count)) != "2" {
		t.Fatalf("agent invocations=%q want 2", count)
	}
	session, err := os.ReadFile(filepath.Join(dir, ".workbench-fake-session"))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := strings.TrimSpace(string(session)), openClawOperationSessionID(task); got != want {
		t.Fatalf("job session=%q want %q", got, want)
	}
	archive, err := os.ReadFile(filepath.Join(dir, ".workbench-fake-archive"))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := strings.TrimSpace(string(archive)), openClawOperationSessionKey(task); got != want {
		t.Fatalf("archived session=%q want %q", got, want)
	}
	if strings.Contains(res.Output, operationCompletePrefix) || !strings.Contains(res.Output, "health verification complete") {
		t.Fatalf("unexpected final report: %q", res.Output)
	}
}
