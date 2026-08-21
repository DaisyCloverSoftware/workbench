//go:build !windows

package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGovernancePatchEquivalentInventoryDistinguishesReviewedAndNovelRefs(t *testing.T) {
	base := t.TempDir()
	remote := filepath.Join(base, "remote.git")
	seed := filepath.Join(base, "seed")
	target := filepath.Join(base, "target")
	patchGitRun(t, base, "init", "--bare", remote)
	if err := os.MkdirAll(seed, 0o755); err != nil {
		t.Fatal(err)
	}
	patchGitRun(t, seed, "init", "-b", "main")
	patchGitRun(t, seed, "config", "user.email", "test@example.invalid")
	patchGitRun(t, seed, "config", "user.name", "Workbench Test")
	patchGitRun(t, seed, "remote", "add", "origin", remote)
	patchWrite(t, filepath.Join(seed, "base.txt"), "base\n")
	patchGitRun(t, seed, "add", ".")
	patchGitRun(t, seed, "commit", "-m", "base")
	baseHead := patchGitOutput(t, seed, "rev-parse", "HEAD")
	patchGitRun(t, seed, "push", "-u", "origin", "main")

	patchGitRun(t, seed, "checkout", "-b", "reviewed-equivalent", baseHead)
	patchWrite(t, filepath.Join(seed, "same.txt"), "same change\n")
	patchGitRun(t, seed, "add", "same.txt")
	patchGitRun(t, seed, "commit", "-m", "reviewed equivalent")
	reviewedTip := patchGitOutput(t, seed, "rev-parse", "HEAD")
	patchGitRun(t, seed, "push", "origin", "reviewed-equivalent")
	patchGitRun(t, remote, "update-ref", "refs/pull/7/head", reviewedTip)

	patchGitRun(t, seed, "checkout", "main")
	# Create the same patch independently on main so git cherry marks the branch
	# commit as patch-equivalent rather than ancestor-equivalent.
	patchWrite(t, filepath.Join(seed, "same.txt"), "same change\n")
	patchGitRun(t, seed, "add", "same.txt")
	patchGitRun(t, seed, "commit", "-m", "same change landed independently")
	patchGitRun(t, seed, "push", "origin", "main")

	patchGitRun(t, seed, "checkout", "-b", "novel", "main")
	patchWrite(t, filepath.Join(seed, "novel.txt"), "novel\n")
	patchGitRun(t, seed, "add", "novel.txt")
	patchGitRun(t, seed, "commit", "-m", "novel")
	patchGitRun(t, seed, "push", "origin", "novel")

	patchGitRun(t, base, "clone", "-b", "main", remote, target)
	script := filepath.Join("..", "..", "scripts", "ops", "governance-patch-equivalent-branch-inventory.sh")
	cmd := exec.Command("bash", script, target)
	cmd.Env = append(os.Environ(), "WORKBENCH_GOVERNANCE_TEST_ALLOW_LOCAL_ORIGIN=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("inventory failed: %v\n%s", err, out)
	}
	text := string(out)
	if !strings.Contains(text, "branch_total=2") || !strings.Contains(text, "patch_equivalent=1") || !strings.Contains(text, "novel_patches=1") {
		t.Fatalf("unexpected totals: %s", text)
	}
	if !strings.Contains(text, "name=reviewed-equivalent") || !strings.Contains(text, "patch_status=patch-equivalent") || !strings.Contains(text, "provenance=pr-head") || !strings.Contains(text, "pr_numbers=7") {
		t.Fatalf("reviewed equivalent branch not classified: %s", text)
	}
	if !strings.Contains(text, "name=novel") || !strings.Contains(text, "patch_status=novel-patches") || !strings.Contains(text, "provenance=protected") {
		t.Fatalf("novel branch not protected: %s", text)
	}
	if status := patchGitOutput(t, target, "status", "--porcelain=v1", "--untracked-files=all"); status != "" {
		t.Fatalf("inventory dirtied checkout: %q", status)
	}
}

func patchWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func patchGitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func patchGitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}
