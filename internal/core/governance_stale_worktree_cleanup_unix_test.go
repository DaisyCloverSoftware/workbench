//go:build !windows

package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type staleWorktreeFixture struct {
	root          string
	seed          string
	target        string
	namedDir      string
	oldHead       string
	desiredHead   string
	namedBranch   string
	auditBranch   string
	secondaryDirs []string
}

func TestGovernanceStaleWorktreeCleanupRemovesOnlyExpectedCleanTopology(t *testing.T) {
	fx := newStaleWorktreeFixture(t, false)
	out, err := runStaleWorktreeCleanup(t, fx)
	if err != nil {
		t.Fatalf("cleanup failed: %v\n%s", err, out)
	}
	text := string(out)
	if !strings.Contains(text, "cleanup=ok") || !strings.Contains(text, "secondary_worktrees_removed=7") {
		t.Fatalf("unexpected cleanup output: %s", text)
	}
	assertStaleWorktreeCleanupResult(t, fx)
}

func TestGovernanceStaleWorktreeCleanupRestoresExactPublishedDuplicate(t *testing.T) {
	fx := newStaleWorktreeFixture(t, false)
	staleWrite(t, filepath.Join(fx.namedDir, "internal/core/changeset_prepare.go"), "published duplicate prepare\n")
	staleWrite(t, filepath.Join(fx.namedDir, "internal/core/changeset_prepare_test.go"), "published duplicate test\n")
	prepareBlob := staleGitOutput(t, fx.namedDir, "hash-object", "internal/core/changeset_prepare.go")
	prepareTestBlob := staleGitOutput(t, fx.namedDir, "hash-object", "internal/core/changeset_prepare_test.go")

	out, err := runStaleWorktreeCleanup(t, fx,
		"WORKBENCH_GOVERNANCE_TEST_EXPECTED_PREPARE_BLOB="+prepareBlob,
		"WORKBENCH_GOVERNANCE_TEST_EXPECTED_PREPARE_TEST_BLOB="+prepareTestBlob,
	)
	if err != nil {
		t.Fatalf("published-duplicate cleanup failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "published_duplicate_restored=1") {
		t.Fatalf("cleanup did not report duplicate restoration: %s", out)
	}
	assertStaleWorktreeCleanupResult(t, fx)
}

func TestGovernanceStaleWorktreeCleanupRefusesDirtyDetached(t *testing.T) {
	fx := newStaleWorktreeFixture(t, true)
	out, err := runStaleWorktreeCleanup(t, fx)
	if err == nil {
		t.Fatalf("dirty detached worktree unexpectedly cleaned: %s", out)
	}
	if !strings.Contains(string(out), "error=detached-worktree-dirty") {
		t.Fatalf("unexpected refusal: %s", out)
	}
	refs := staleGitOutput(t, fx.target, "worktree", "list", "--porcelain")
	if strings.Count(refs, "worktree ") != 8 {
		t.Fatalf("refused cleanup changed worktree topology: %s", refs)
	}
	if got := staleGitOutput(t, fx.target, "rev-parse", "HEAD"); got != fx.oldHead {
		t.Fatalf("refused cleanup moved main: %s", got)
	}
}

func assertStaleWorktreeCleanupResult(t *testing.T, fx staleWorktreeFixture) {
	t.Helper()
	if got := staleGitOutput(t, fx.target, "rev-parse", "HEAD"); got != fx.desiredHead {
		t.Fatalf("main head=%s want=%s", got, fx.desiredHead)
	}
	if got := staleGitOutput(t, fx.target, "rev-parse", "refs/heads/"+fx.auditBranch); got != fx.oldHead {
		t.Fatalf("audit branch=%s want=%s", got, fx.oldHead)
	}
	if refs := staleGitOutput(t, fx.target, "worktree", "list", "--porcelain"); strings.Count(refs, "worktree ") != 1 {
		t.Fatalf("secondary worktrees remain: %s", refs)
	}
	cmd := exec.Command("git", "-C", fx.target, "show-ref", "--verify", "--quiet", "refs/heads/"+fx.namedBranch)
	if err := cmd.Run(); err == nil {
		t.Fatalf("named stale branch %q still exists", fx.namedBranch)
	}
	if status := staleGitOutput(t, fx.target, "status", "--porcelain=v1", "--untracked-files=all"); status != "" {
		t.Fatalf("target remains dirty: %q", status)
	}
}

func newStaleWorktreeFixture(t *testing.T, dirtyDetached bool) staleWorktreeFixture {
	t.Helper()
	base := t.TempDir()
	remote := filepath.Join(base, "remote.git")
	seed := filepath.Join(base, "seed")
	target := filepath.Join(base, "target")
	staleGitRun(t, base, "init", "--bare", remote)
	if err := os.MkdirAll(seed, 0o755); err != nil {
		t.Fatal(err)
	}
	staleGitRun(t, seed, "init", "-b", "main")
	staleGitRun(t, seed, "config", "user.email", "test@example.invalid")
	staleGitRun(t, seed, "config", "user.name", "Workbench Test")
	staleGitRun(t, seed, "remote", "add", "origin", remote)
	staleWrite(t, filepath.Join(seed, "base.txt"), "base\n")
	if err := os.MkdirAll(filepath.Join(seed, "internal", "core"), 0o755); err != nil {
		t.Fatal(err)
	}
	staleWrite(t, filepath.Join(seed, "internal/core/changeset_prepare.go"), "base prepare\n")
	staleWrite(t, filepath.Join(seed, "internal/core/changeset_prepare_test.go"), "base test\n")
	staleGitRun(t, seed, "add", ".")
	staleGitRun(t, seed, "commit", "-m", "base")
	oldHead := staleGitOutput(t, seed, "rev-parse", "HEAD")
	staleGitRun(t, seed, "push", "-u", "origin", "main")
	staleGitRun(t, base, "clone", "-b", "main", remote, target)
	staleGitRun(t, target, "config", "user.email", "test@example.invalid")
	staleGitRun(t, target, "config", "user.name", "Workbench Test")

	auditBranch := "audit/test-pre-reset"
	namedBranch := "fix/test-stale-worktree"
	staleGitRun(t, target, "branch", auditBranch, oldHead)
	var secondary []string
	for i := 0; i < 6; i++ {
		path := filepath.Join(base, "detached-"+string(rune('a'+i)))
		staleGitRun(t, target, "worktree", "add", "--detach", path, oldHead)
		secondary = append(secondary, path)
	}
	namedPath := filepath.Join(base, "named")
	staleGitRun(t, target, "worktree", "add", "-b", namedBranch, namedPath, oldHead)
	secondary = append(secondary, namedPath)
	if dirtyDetached {
		staleWrite(t, filepath.Join(secondary[0], "dirty.txt"), "dirty\n")
	}

	staleWrite(t, filepath.Join(seed, "current.txt"), "current\n")
	staleGitRun(t, seed, "add", "current.txt")
	staleGitRun(t, seed, "commit", "-m", "current main")
	desired := staleGitOutput(t, seed, "rev-parse", "HEAD")
	staleGitRun(t, seed, "push", "origin", "main")

	return staleWorktreeFixture{
		root:          base,
		seed:          seed,
		target:        target,
		namedDir:      namedPath,
		oldHead:       oldHead,
		desiredHead:   desired,
		namedBranch:   namedBranch,
		auditBranch:   auditBranch,
		secondaryDirs: secondary,
	}
}

func runStaleWorktreeCleanup(t *testing.T, fx staleWorktreeFixture, extraEnv ...string) ([]byte, error) {
	t.Helper()
	script := filepath.Join("..", "..", "scripts", "ops", "governance-clean-stale-worktrees.sh")
	cmd := exec.Command("bash", script, fx.target)
	cmd.Env = append(os.Environ(),
		"WORKBENCH_GOVERNANCE_TEST_ALLOW_LOCAL_ORIGIN=1",
		"WORKBENCH_GOVERNANCE_TEST_EXPECTED_MAIN="+fx.oldHead,
		"WORKBENCH_GOVERNANCE_TEST_EXPECTED_DETACHED="+fx.oldHead,
		"WORKBENCH_GOVERNANCE_TEST_EXPECTED_DETACHED_COUNT=6",
		"WORKBENCH_GOVERNANCE_TEST_EXPECTED_NAMED_HEAD="+fx.oldHead,
		"WORKBENCH_GOVERNANCE_TEST_EXPECTED_NAMED_BRANCH="+fx.namedBranch,
		"WORKBENCH_GOVERNANCE_TEST_AUDIT_BRANCH="+fx.auditBranch,
		"WORKBENCH_OPERATION_COMMIT="+fx.desiredHead,
	)
	cmd.Env = append(cmd.Env, extraEnv...)
	return cmd.CombinedOutput()
}

func staleWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func staleGitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func staleGitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}
