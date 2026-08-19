//go:build !windows

package core

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOperationsContextOverflowFailsOverToClaudeInSameJobConversation(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "fake-openclaw")
	body := `#!/usr/bin/env bash
set -euo pipefail
script_dir="$(cd "$(dirname "$0")" && pwd)"
if [ "${1:-}" = "models" ] && [ "${2:-}" = "status" ]; then
  echo '{"resolvedDefault":"openai/gpt-5.3-codex-spark"}'
  exit 0
fi
if [ "${1:-}" = "models" ] && [ "${2:-}" = "list" ]; then
  echo '{"models":[{"key":"openai/gpt-5.3-codex-spark","provider":"openai","available":true},{"key":"anthropic/claude-opus","provider":"anthropic","available":true},{"key":"anthropic/claude-sonnet","provider":"anthropic","available":true}]}'
  exit 0
fi
if [ "${1:-}" = "sessions" ] && [ "${2:-}" = "archive" ]; then
  printf '%s' "${3:-}" > "$script_dir/archive-key"
  exit 0
fi
session=""
model=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --session-id) session="${2:-}"; shift 2 ;;
    --model) model="${2:-}"; shift 2 ;;
    *) shift ;;
  esac
done
if [ -z "$session" ]; then echo 'missing session' >&2; exit 2; fi
if [ -f "$script_dir/session-id" ] && [ "$(cat "$script_dir/session-id")" != "$session" ]; then
  echo 'model fallback changed job conversation' >&2
  exit 3
fi
printf '%s' "$session" > "$script_dir/session-id"
if [ -z "$model" ]; then
  echo 'Context overflow: prompt too large for the model. Try /reset (or /new) to start a fresh session, or use a larger-context model.'
  exit 1
fi
printf '%s' "$model" > "$script_dir/fallback-model"
if [ "$model" != "anthropic/claude-sonnet" ]; then
  echo "unexpected model $model" >&2
  exit 4
fi
echo 'Claude verified the read-only machine operation.'
echo 'WORKBENCH_OPERATION_COMPLETE: verified'
`
	if err := os.WriteFile(script, []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}

	provider := Provider{ID: "openclaw", Name: "OpenClaw", Command: script, Installed: true, Authenticated: true, CanWrite: true, CanRunTools: true}
	task := Task{ID: "task-context-failover", Mode: TaskModeOperations, ProjectPath: dir, Intent: "Verify read-only host health"}
	res, err := RunOpenClawOperationSupervised(context.Background(), provider, task, Preferences{})
	if err != nil {
		t.Fatalf("supervised context failover failed: %v; output=%q", err, res.Output)
	}
	model, err := os.ReadFile(filepath.Join(dir, "fallback-model"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(model)) != "anthropic/claude-sonnet" {
		t.Fatalf("fallback model=%q", model)
	}
	session, err := os.ReadFile(filepath.Join(dir, "session-id"))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := strings.TrimSpace(string(session)), openClawOperationSessionID(task); got != want {
		t.Fatalf("session=%q want %q", got, want)
	}
	archive, err := os.ReadFile(filepath.Join(dir, "archive-key"))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := strings.TrimSpace(string(archive)), openClawOperationSessionKey(task); got != want {
		t.Fatalf("archive=%q want %q", got, want)
	}
	if !strings.Contains(res.Output, "Claude verified") {
		t.Fatalf("final report=%q", res.Output)
	}
}
