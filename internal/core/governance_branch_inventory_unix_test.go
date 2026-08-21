//go:build !windows

package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGovernanceBranchInventoryClassifiesMergedAndDivergedRefs(t *testing.T) {
	base := t.TempDir()
	remote := filepath.Join(base, "remote.git")
	seed := filepath.Join(base, "seed")
	target := filepath.Join(base, "target")
	branchGitRun(t, base, "init", "--bare", remote)
	if err := os.MkdirAll(seed, 0o755); err != nil {
		t.Fatal(err)
	}
	branchGitRun(t, seed, "init", "-b", "main")
	branchGitRun(t, seed, "config", "user.email", "test@example.invalid")
	branchGitRun(t, seed, "config", "user.name", "Workbench Test")
	branchGitRun(t, seed, "remote", "add", "origin", remote)
	branchWrite(t, filepath.Join(seed, "base.txt"), "base\n")
	branchGitRun(t, seed, "add", ".")
	branchGitRun(t, seed, "commit", "-m", "base")
	branchGitRun(t, seed, "branch", "merged-old")
	branchGitRun(t, seed, "push", "origin", "main", "merged-old")

	branchWrite(t, filepath.Join(seed, "main.txt"), "main\n")
	branchGitRun(t, seed, "add", "main.txt")
	branchGitRun(t, seed, "commit", "-m", "main advance")
	branchGitRun(t, seed, "push", "origin", "main")

	branchGitRun(t, seed, "checkout", "-b", "diverged", "HEAD~1")
	branchWrite(t, filepath.Join(seed, "diverged.txt"), "diverged\n")
	branchGitRun(t, seed, "add", "diverged.txt")
	branchGitRun(t, seed, "commit", "-m", "diverged change")
	branchGitRun(t, seed, "push", "origin", "diverged")

	branchGitRun(t, base, "clone", remote, target)
	script := filepath.Join("..", "..", "scripts", "ops", "governance-branch-inventory.sh")
	cmd := exec.Command("bash", script, target)
	cmd.Env = append(os.Environ(), "WORKBENCH_GOVERNANCE_TEST_ALLOW_LOCAL_ORIGIN=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("inventory failed: %v\n%s", err, out)
	}
	text := string(out)
	if !strings.Contains(text, "branch_total=3") {
		t.Fatalf("unexpected total: %s", text)
	}
	if !strings.Contains(text, "name=merged-old") || !strings.Contains(text, "status=merged") {
		t.Fatalf("merged branch not classified: %s", text)
	}
	if !strings.Contains(text, "name=diverged") || !strings.Contains(text, "status=diverged") || !strings.Contains(text, "unique_commits=1") {
		t.Fatalf("diverged branch not classified: %s", text)
	}
	if status := branchGitOutput(t, target, "status", "--porcelain=v1", "--untracked-files=all"); status != "" {
		t.Fatalf("inventory dirtied checkout: %q", status)
	}
}

func branchWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func branchGitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func branchGitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}
