//go:build !windows

package core

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunProviderClaudeCapturesAndResumesSameTaskSession(t *testing.T) {
	isolateProviderSessions(t)
	dir := t.TempDir()
	logPath := filepath.Join(dir, "calls.log")
	claude := filepath.Join(dir, "fake-claude")
	sessionID := "550e8400-e29b-41d4-a716-446655440000"
	script := `#!/bin/sh
set -eu
case " $* " in
  *" --resume ` + sessionID + ` "*)
    printf '%s\n' 'resume:` + sessionID + `' >> "$CLAUDE_CALL_LOG"
    printf '%s\n' '{"type":"result","subtype":"success","result":"resumed completion","session_id":"` + sessionID + `"}'
    ;;
  *)
    printf '%s\n' 'fresh' >> "$CLAUDE_CALL_LOG"
    printf '%s\n' '{"type":"result","subtype":"success","result":"fresh completion","session_id":"` + sessionID + `"}'
    ;;
esac
`
	if err := os.WriteFile(claude, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAUDE_CALL_LOG", logPath)
	project := t.TempDir()
	task := Task{ID: "task-claude-resume", ProjectPath: project, Intent: "Finish the implementation"}
	provider := Provider{ID: "claude", Name: "Claude", Command: claude, Installed: true, Authenticated: true, Cost: CostIncluded, CanWrite: true}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	first, err := RunProvider(ctx, provider, task, Preferences{})
	if err != nil {
		t.Fatal(err)
	}
	if first.Output != "fresh completion" {
		t.Fatalf("first output=%q", first.Output)
	}
	session, ok, err := ProviderSessionFor(task.ID, provider.ID)
	if err != nil || !ok || session.SessionID != sessionID {
		t.Fatalf("captured session=%#v ok=%t err=%v", session, ok, err)
	}

	second, err := RunProvider(ctx, provider, task, Preferences{})
	if err != nil {
		t.Fatal(err)
	}
	if second.Output != "resumed completion" {
		t.Fatalf("second output=%q", second.Output)
	}
	calls, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(calls)), "\n")
	if len(lines) != 2 {
		t.Fatalf("Claude invocation count=%d want 2: %s", len(lines), calls)
	}
	if lines[0] != "fresh" {
		t.Fatalf("first Claude invocation unexpectedly resumed: %s", lines[0])
	}
	if lines[1] != "resume:"+sessionID {
		t.Fatalf("second Claude invocation did not resume exact task session: %s", lines[1])
	}
}

func TestRunProviderClaudeFallsBackFreshOnceWhenStoredSessionIsGone(t *testing.T) {
	isolateProviderSessions(t)
	dir := t.TempDir()
	logPath := filepath.Join(dir, "calls.log")
	claude := filepath.Join(dir, "fake-claude")
	staleID := "550e8400-e29b-41d4-a716-446655440000"
	freshID := "123e4567-e89b-12d3-a456-426614174000"
	script := `#!/bin/sh
set -eu
case " $* " in
  *" --resume ` + staleID + ` "*)
    printf '%s\n' 'resume:` + staleID + `' >> "$CLAUDE_CALL_LOG"
    printf '%s\n' 'Session not found' >&2
    exit 2
    ;;
  *)
    printf '%s\n' 'fresh' >> "$CLAUDE_CALL_LOG"
    printf '%s\n' '{"type":"result","subtype":"success","result":"fresh fallback completed","session_id":"` + freshID + `"}'
    ;;
esac
`
	if err := os.WriteFile(claude, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAUDE_CALL_LOG", logPath)
	project := t.TempDir()
	task := Task{ID: "task-claude-stale", ProjectPath: project, Intent: "Finish safely"}
	provider := Provider{ID: "claude", Name: "Claude", Command: claude, Installed: true, Authenticated: true, Cost: CostIncluded, CanWrite: true}
	if _, err := SaveProviderSession(task.ID, provider.ID, staleID); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := RunProvider(ctx, provider, task, Preferences{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Output != "fresh fallback completed" {
		t.Fatalf("fallback output=%q", res.Output)
	}
	session, ok, err := ProviderSessionFor(task.ID, provider.ID)
	if err != nil || !ok || session.SessionID != freshID {
		t.Fatalf("fresh session was not stored: %#v ok=%t err=%v", session, ok, err)
	}
	calls, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(calls)), "\n")
	if len(lines) != 2 {
		t.Fatalf("stale-session fallback invoked Claude %d times, want exactly 2: %s", len(lines), calls)
	}
	if lines[0] != "resume:"+staleID || lines[1] != "fresh" {
		t.Fatalf("unexpected stale-session fallback sequence: %#v", lines)
	}
}
