//go:build !windows

package core

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestOperationsLaneDiscardsOpenClawSourceEdits(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	if out, err := exec.Command("git", "init", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	for _, args := range [][]string{{"config", "user.email", "workbench-test@example.invalid"}, {"config", "user.name", "Workbench Test"}} {
		if out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git config: %v: %s", err, out)
		}
	}
	original := "original\n"
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", repo, "add", "README.md").CombinedOutput(); err != nil {
		t.Fatalf("git add: %v: %s", err, out)
	}
	if out, err := exec.Command("git", "-C", repo, "commit", "-m", "fixture").CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v: %s", err, out)
	}

	script := filepath.Join(t.TempDir(), "fake-openclaw")
	body := `#!/usr/bin/env bash
set -euo pipefail
printf 'changed by operator\n' > README.md
echo 'Operational action complete.'
echo 'WORKBENCH_OPERATION_COMPLETE: verified'
`
	if err := os.WriteFile(script, []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
	provider := Provider{ID: "openclaw", Name: "OpenClaw", Command: script, Installed: true, Authenticated: true, CanWrite: true, CanRunTools: true}
	task := Task{ID: "task-op-boundary", Mode: TaskModeOperations, ProjectPath: repo, Intent: "Restart a service; do not change source"}
	res, err := RunProviderIsolated(context.Background(), provider, task, Preferences{})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "source-code boundary") {
		t.Fatalf("source edit was not rejected: err=%v output=%q", err, res.Output)
	}
	bodyAfter, readErr := os.ReadFile(filepath.Join(repo, "README.md"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(bodyAfter) != original {
		t.Fatalf("original checkout changed: %q", bodyAfter)
	}
	status, statusErr := exec.Command("git", "-C", repo, "status", "--porcelain").Output()
	if statusErr != nil {
		t.Fatal(statusErr)
	}
	if strings.TrimSpace(string(status)) != "" {
		t.Fatalf("original checkout became dirty: %q", status)
	}
}
