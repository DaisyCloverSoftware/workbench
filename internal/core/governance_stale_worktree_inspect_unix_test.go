//go:build !windows

package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGovernanceStaleWorktreeInspectionReturnsOnlyExpectedRepositoryDiff(t *testing.T) {
	base := t.TempDir()
	repo := filepath.Join(base, "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	staleGitRun(t, repo, "init", "-b", "main")
	staleGitRun(t, repo, "config", "user.email", "test@example.invalid")
	staleGitRun(t, repo, "config", "user.name", "Workbench Test")
	staleGitRun(t, repo, "remote", "add", "origin", "https://github.com/DaisyCloverSoftware/workbench.git")
	staleWrite(t, filepath.Join(repo, "internal/core/changeset_prepare.go"), "package core\n\nvar value = 1\n")
	staleWrite(t, filepath.Join(repo, "internal/core/changeset_prepare_test.go"), "package core\n\nvar testValue = 1\n")
	staleGitRun(t, repo, "add", ".")
	staleGitRun(t, repo, "commit", "-m", "base")
	head := staleGitOutput(t, repo, "rev-parse", "HEAD")

	branch := "fix/test-file-modes"
	other := filepath.Join(base, "stale")
	staleGitRun(t, repo, "worktree", "add", "-b", branch, other, head)
	staleWrite(t, filepath.Join(other, "internal/core/changeset_prepare.go"), "package core\n\nvar value = 2\n")
	staleWrite(t, filepath.Join(other, "internal/core/changeset_prepare_test.go"), "package core\n\nvar testValue = 2\n")

	script := filepath.Join("..", "..", "scripts", "ops", "governance-inspect-stale-worktree.sh")
	cmd := exec.Command("bash", script, repo)
	cmd.Env = append(os.Environ(),
		"WORKBENCH_GOVERNANCE_TEST_ALLOW_LOCAL_ORIGIN=1",
		"WORKBENCH_GOVERNANCE_TEST_EXPECTED_BRANCH="+branch,
		"WORKBENCH_GOVERNANCE_TEST_EXPECTED_HEAD="+head,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("inspection failed: %v\n%s", err, out)
	}
	text := string(out)
	if !strings.Contains(text, "inspection=ok") || !strings.Contains(text, "-var value = 1") || !strings.Contains(text, "+var value = 2") {
		t.Fatalf("inspection missing expected diff: %s", text)
	}
	if strings.Contains(text, repo) || strings.Contains(text, other) {
		t.Fatalf("inspection leaked worktree path: %s", text)
	}
}
