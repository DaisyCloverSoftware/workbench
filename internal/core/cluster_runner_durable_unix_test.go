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

func installFakeRunnerSSH(t *testing.T, body string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "calls.log")
	script := filepath.Join(dir, "ssh")
	content := "#!/bin/sh\nset -eu\n" + body + "\n"
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKE_SSH_LOG", logPath)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return script, logPath
}

func durableSSHTestTask() Task {
	return Task{ID: "task-ssh-durable", Intent: "Do durable work", ProjectPath: "/desktop/project"}
}

func TestRunClusterRunnerSSHRetriesStatusWithoutResubmittingWork(t *testing.T) {
	_, logPath := installFakeRunnerSSH(t, `
printf '%s\n' "$*" >> "$FAKE_SSH_LOG"
case "$*" in
  *"job submit")
    cat >/dev/null
    printf '%s\n' '{"ok":true,"result":{"job":{"id":"task-ssh-durable","task_id":"task-ssh-durable","generation":1,"status":"queued","created_at":"2026-08-15T00:00:00Z","updated_at":"2026-08-15T00:00:00Z"},"reused":false}}'
    ;;
  *"job status task-ssh-durable")
    count_file="${FAKE_SSH_LOG}.status"
    count=0
    [ ! -f "$count_file" ] || count=$(cat "$count_file")
    count=$((count + 1))
    printf '%s' "$count" > "$count_file"
    if [ "$count" -eq 1 ]; then
      exit 255
    fi
    printf '%s\n' '{"ok":true,"job":{"id":"task-ssh-durable","task_id":"task-ssh-durable","generation":1,"status":"completed","created_at":"2026-08-15T00:00:00Z","updated_at":"2026-08-15T00:00:01Z","response":{"result":{"output":"durable done"},"provider_id":"claude","provider_name":"Claude"}}}'
    ;;
  *) exit 2 ;;
esac`)

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	res, err := RunClusterRunnerSSH(ctx, "runner.example", durableSSHTestTask(), Preferences{AvoidWorkUsage: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Output != "durable done" || res.WorkerProviderID != "claude" {
		t.Fatalf("unexpected durable result: %+v", res)
	}
	calls, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(calls)
	if strings.Count(text, "job submit") != 1 {
		t.Fatalf("durable work was resubmitted after status transport failure:\n%s", text)
	}
	if strings.Count(text, "job status task-ssh-durable") < 2 {
		t.Fatalf("status was not retried after transport failure:\n%s", text)
	}
}

func TestRunClusterRunnerSSHExplicitCancellationCancelsRemoteJob(t *testing.T) {
	_, logPath := installFakeRunnerSSH(t, `
printf '%s\n' "$*" >> "$FAKE_SSH_LOG"
case "$*" in
  *"job submit")
    cat >/dev/null
    printf '%s\n' '{"ok":true,"result":{"job":{"id":"task-ssh-durable","task_id":"task-ssh-durable","generation":1,"status":"running","created_at":"2026-08-15T00:00:00Z","updated_at":"2026-08-15T00:00:00Z"},"reused":false}}'
    ;;
  *"job status task-ssh-durable") exit 255 ;;
  *"job cancel task-ssh-durable")
    printf '%s\n' '{"ok":true,"job":{"id":"task-ssh-durable","task_id":"task-ssh-durable","generation":1,"status":"cancelled","created_at":"2026-08-15T00:00:00Z","updated_at":"2026-08-15T00:00:01Z"}}'
    ;;
  *) exit 2 ;;
esac`)

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	_, err := RunClusterRunnerSSH(ctx, "runner.example", durableSSHTestTask(), Preferences{AvoidWorkUsage: true})
	if err == nil || !strings.Contains(err.Error(), "deadline") {
		t.Fatalf("expected local cancellation/deadline, got %v", err)
	}
	calls, readErr := os.ReadFile(logPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !strings.Contains(string(calls), "job cancel task-ssh-durable") {
		t.Fatalf("remote durable job was not cancelled after explicit local cancellation:\n%s", calls)
	}
}
