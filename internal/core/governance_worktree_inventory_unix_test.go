//go:build !windows

package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGovernanceWorktreeInventoryOmitsPaths(t *testing.T) {
	repo := t.TempDir()
	govGitRun(t, repo, "init", "-b", "main")
	govGitRun(t, repo, "config", "user.email", "test@example.invalid")
	govGitRun(t, repo, "config", "user.name", "Workbench Test")
	govGitRun(t, repo, "remote", "add", "origin", "https://github.com/DaisyCloverSoftware/workbench.git")
	if err := os.WriteFile(filepath.Join(repo, "tracked.txt"), []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	govGitRun(t, repo, "add", "tracked.txt")
	govGitRun(t, repo, "commit", "-m", "fixture")

	other := filepath.Join(t.TempDir(), "secondary")
	govGitRun(t, repo, "worktree", "add", "-b", "audit-secondary", other)

	script := filepath.Join("..", "..", "scripts", "ops", "governance-worktree-inventory.sh")
	cmd := exec.Command("bash", script, repo)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("inventory failed: %v\n%s", err, out)
	}
	text := string(out)
	if !strings.Contains(text, "worktree_count=2") {
		t.Fatalf("unexpected inventory: %s", text)
	}
	if !strings.Contains(text, "worktree_1_branch=main") || !strings.Contains(text, "worktree_2_branch=audit-secondary") {
		t.Fatalf("inventory missing branches: %s", text)
	}
	if strings.Contains(text, repo) || strings.Contains(text, other) {
		t.Fatalf("inventory leaked filesystem path: %s", text)
	}
}

func govGitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}
