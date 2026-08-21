//go:build !windows

package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGovernanceRemoteBranchCleanupDeletesOnlyMergedRefs(t *testing.T) {
	base := t.TempDir()
	remote := filepath.Join(base, "remote.git")
	seed := filepath.Join(base, "seed")
	target := filepath.Join(base, "target")
	remoteGitRun(t, base, "init", "--bare", remote)
	if err := os.MkdirAll(seed, 0o755); err != nil {
		t.Fatal(err)
	}
	remoteGitRun(t, seed, "init", "-b", "main")
	remoteGitRun(t, seed, "config", "user.email", "test@example.invalid")
	remoteGitRun(t, seed, "config", "user.name", "Workbench Test")
	remoteGitRun(t, seed, "remote", "add", "origin", remote)
	remoteWrite(t, filepath.Join(seed, "base.txt"), "base\n")
	remoteGitRun(t, seed, "add", ".")
	remoteGitRun(t, seed, "commit", "-m", "base")
	baseHead := remoteGitOutput(t, seed, "rev-parse", "HEAD")
	remoteGitRun(t, seed, "branch", "merged-old")
	remoteGitRun(t, seed, "push", "origin", "main", "merged-old")

	remoteWrite(t, filepath.Join(seed, "main.txt"), "main\n")
	remoteGitRun(t, seed, "add", "main.txt")
	remoteGitRun(t, seed, "commit", "-m", "main advance")
	desired := remoteGitOutput(t, seed, "rev-parse", "HEAD")
	remoteGitRun(t, seed, "push", "origin", "main")

	remoteGitRun(t, seed, "checkout", "-b", "diverged", baseHead)
	remoteWrite(t, filepath.Join(seed, "diverged.txt"), "diverged\n")
	remoteGitRun(t, seed, "add", "diverged.txt")
	remoteGitRun(t, seed, "commit", "-m", "diverged")
	remoteGitRun(t, seed, "push", "origin", "diverged")

	remoteGitRun(t, base, "clone", "-b", "main", remote, target)
	script := filepath.Join("..", "..", "scripts", "ops", "governance-delete-merged-remote-branches.sh")
	cmd := exec.Command("bash", script, target, "confirmed-no-open-prs")
	cmd.Env = append(os.Environ(),
		"WORKBENCH_GOVERNANCE_TEST_ALLOW_LOCAL_ORIGIN=1",
		"WORKBENCH_OPERATION_COMMIT="+desired,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cleanup failed: %v\n%s", err, out)
	}
	text := string(out)
	if !strings.Contains(text, "cleanup=ok") || !strings.Contains(text, "deleted=1") || !strings.Contains(text, "diverged_retained=1") {
		t.Fatalf("unexpected cleanup output: %s", text)
	}
	if remoteRefExists(t, remote, "refs/heads/merged-old") {
		t.Fatal("fully merged remote branch was not deleted")
	}
	if !remoteRefExists(t, remote, "refs/heads/diverged") {
		t.Fatal("diverged remote branch was incorrectly deleted")
	}
	if !remoteRefExists(t, remote, "refs/heads/main") {
		t.Fatal("main was incorrectly deleted")
	}
	if status := remoteGitOutput(t, target, "status", "--porcelain=v1", "--untracked-files=all"); status != "" {
		t.Fatalf("cleanup dirtied checkout: %q", status)
	}
}

func TestGovernanceRemoteBranchCleanupRequiresExplicitNoOpenPRConfirmation(t *testing.T) {
	repo := t.TempDir()
	script := filepath.Join("..", "..", "scripts", "ops", "governance-delete-merged-remote-branches.sh")
	cmd := exec.Command("bash", script, repo, "not-confirmed")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("cleanup accepted missing confirmation: %s", out)
	}
	if !strings.Contains(string(out), "error=no-open-pr-confirmation-required") {
		t.Fatalf("unexpected refusal: %s", out)
	}
}

func remoteRefExists(t *testing.T, bareRepo, ref string) bool {
	t.Helper()
	cmd := exec.Command("git", "-C", bareRepo, "show-ref", "--verify", "--quiet", ref)
	err := cmd.Run()
	if err == nil {
		return true
	}
	if _, ok := err.(*exec.ExitError); ok {
		return false
	}
	t.Fatalf("show-ref %s failed: %v", ref, err)
	return false
}

func remoteWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func remoteGitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func remoteGitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}
