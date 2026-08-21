//go:build !windows

package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGovernanceArchiveHistoricalBranchesPreservesEveryTipAndCanonicalTree(t *testing.T) {
	base := t.TempDir()
	remote := filepath.Join(base, "remote.git")
	seed := filepath.Join(base, "seed")
	target := filepath.Join(base, "target")
	archiveGitRun(t, base, "init", "--bare", remote)
	if err := os.MkdirAll(seed, 0o755); err != nil {
		t.Fatal(err)
	}
	archiveGitRun(t, seed, "init", "-b", "main")
	archiveGitRun(t, seed, "config", "user.email", "test@example.invalid")
	archiveGitRun(t, seed, "config", "user.name", "Workbench Test")
	archiveGitRun(t, seed, "remote", "add", "origin", remote)
	archiveWrite(t, filepath.Join(seed, "base.txt"), "base\n")
	archiveGitRun(t, seed, "add", ".")
	archiveGitRun(t, seed, "commit", "-m", "base")
	baseHead := archiveGitOutput(t, seed, "rev-parse", "HEAD")
	archiveGitRun(t, seed, "push", "-u", "origin", "main")

	// Create several branch histories that are intentionally not all reachable
	// from main. The archive operation must preserve them without merging their
	// trees into canonical main.
	branchTips := map[string]string{}
	for _, branch := range []string{"old/feature-a", "old/feature-b", "old/diagnostic"} {
		archiveGitRun(t, seed, "checkout", "-B", branch, baseHead)
		archiveWrite(t, filepath.Join(seed, strings.ReplaceAll(branch, "/", "-")+".txt"), branch+"\n")
		archiveGitRun(t, seed, "add", ".")
		archiveGitRun(t, seed, "commit", "-m", branch)
		branchTips[branch] = archiveGitOutput(t, seed, "rev-parse", "HEAD")
		archiveGitRun(t, seed, "push", "origin", branch)
	}

	archiveGitRun(t, seed, "checkout", "main")
	archiveWrite(t, filepath.Join(seed, "current.txt"), "canonical current main\n")
	archiveGitRun(t, seed, "add", "current.txt")
	archiveGitRun(t, seed, "commit", "-m", "current main")
	desired := archiveGitOutput(t, seed, "rev-parse", "HEAD")
	mainTree := archiveGitOutput(t, seed, "rev-parse", desired+"^{tree}")
	archiveGitRun(t, seed, "push", "origin", "main")

	archiveGitRun(t, base, "clone", "-b", "main", remote, target)
	archiveGitRun(t, target, "config", "user.email", "test@example.invalid")
	archiveGitRun(t, target, "config", "user.name", "Workbench Test")

	script := filepath.Join("..", "..", "scripts", "ops", "governance-archive-historical-remote-branches.sh")
	cmd := exec.Command("bash", script, target, "confirmed-no-open-prs")
	cmd.Env = append(os.Environ(),
		"WORKBENCH_GOVERNANCE_TEST_ALLOW_LOCAL_ORIGIN=1",
		"WORKBENCH_OPERATION_COMMIT="+desired,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("archive cleanup failed: %v\n%s", err, out)
	}
	text := string(out)
	if !strings.Contains(text, "cleanup=ok") || !strings.Contains(text, "archived_source_refs=3") || !strings.Contains(text, "remote_branch_count=2") {
		t.Fatalf("unexpected archive output: %s", text)
	}

	archiveRef := "refs/heads/archive/pre-governance-reset-20260821"
	archiveHead := archiveGitOutput(t, remote, "rev-parse", archiveRef)
	archiveTree := archiveGitOutput(t, remote, "rev-parse", archiveHead+"^{tree}")
	if archiveTree != mainTree {
		t.Fatalf("archive tree=%s want canonical main tree=%s", archiveTree, mainTree)
	}
	for branch, tip := range branchTips {
		cmd := exec.Command("git", "-C", remote, "merge-base", "--is-ancestor", tip, archiveHead)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("archived tip %s (%s) is not reachable from archive: %v\n%s", branch, tip, err, out)
		}
		if archiveRemoteRefExists(t, remote, "refs/heads/"+branch) {
			t.Fatalf("source branch %q still exists after archive cleanup", branch)
		}
	}
	if !archiveRemoteRefExists(t, remote, "refs/heads/main") || !archiveRemoteRefExists(t, remote, archiveRef) {
		t.Fatal("main or archive ref missing after cleanup")
	}
	refs := archiveGitOutput(t, remote, "for-each-ref", "--format=%(refname)", "refs/heads/")
	if len(strings.Fields(refs)) != 2 {
		t.Fatalf("unexpected remote refs after cleanup: %s", refs)
	}
	if status := archiveGitOutput(t, target, "status", "--porcelain=v1", "--untracked-files=all"); status != "" {
		t.Fatalf("archive cleanup dirtied checkout: %q", status)
	}
}

func TestGovernanceArchiveHistoricalBranchesRequiresExplicitNoOpenPRConfirmation(t *testing.T) {
	repo := t.TempDir()
	script := filepath.Join("..", "..", "scripts", "ops", "governance-archive-historical-remote-branches.sh")
	cmd := exec.Command("bash", script, repo, "not-confirmed")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("archive cleanup accepted missing confirmation: %s", out)
	}
	if !strings.Contains(string(out), "error=no-open-pr-confirmation-required") {
		t.Fatalf("unexpected refusal: %s", out)
	}
}

func archiveRemoteRefExists(t *testing.T, bareRepo, ref string) bool {
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

func archiveWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func archiveGitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func archiveGitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}
