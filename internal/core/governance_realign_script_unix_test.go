//go:build !windows

package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGovernanceRealignPreservesAuditRefAndMovesMain(t *testing.T) {
	base := t.TempDir()
	remote := filepath.Join(base, "remote.git")
	seed := filepath.Join(base, "seed")
	target := filepath.Join(base, "target")
	govRealignGitRun(t, base, "init", "--bare", remote)
	if err := os.MkdirAll(seed, 0o755); err != nil {
		t.Fatal(err)
	}
	govRealignGitRun(t, seed, "init", "-b", "main")
	govRealignGitRun(t, seed, "config", "user.email", "test@example.invalid")
	govRealignGitRun(t, seed, "config", "user.name", "Workbench Test")
	govRealignGitRun(t, seed, "remote", "add", "origin", remote)
	for _, name := range []string{"build.yml", "release.yml", "runner.yml"} {
		govRealignWrite(t, filepath.Join(seed, ".github", "workflows", name), "uses: action@v1\n")
	}
	govRealignGitRun(t, seed, "add", ".")
	govRealignGitRun(t, seed, "commit", "-m", "base")
	baseHead := govRealignGitOutput(t, seed, "rev-parse", "HEAD")
	govRealignGitRun(t, seed, "push", "-u", "origin", "main")
	govRealignGitRun(t, base, "clone", "-b", "main", remote, target)
	govRealignGitRun(t, target, "config", "user.email", "test@example.invalid")
	govRealignGitRun(t, target, "config", "user.name", "Workbench Test")

	for _, name := range []string{"build.yml", "release.yml", "runner.yml"} {
		govRealignWrite(t, filepath.Join(target, ".github", "workflows", name), "uses: action@0123456789abcdef0123456789abcdef01234567\n")
	}
	govRealignGitRun(t, target, "add", ".github/workflows")
	govRealignGitRun(t, target, "commit", "-m", "sec: pin GitHub Actions to full commit SHAs (SEC-008)")
	localHead := govRealignGitOutput(t, target, "rev-parse", "HEAD")

	govRealignGitRun(t, seed, "reset", "--hard", baseHead)
	govRealignWrite(t, filepath.Join(seed, "current.txt"), "current public main\n")
	govRealignGitRun(t, seed, "add", "current.txt")
	govRealignGitRun(t, seed, "commit", "-m", "current main")
	desired := govRealignGitOutput(t, seed, "rev-parse", "HEAD")
	govRealignGitRun(t, seed, "push", "origin", "main")

	script := filepath.Join("..", "..", "scripts", "ops", "governance-realign-workbench-checkout.sh")
	cmd := exec.Command("bash", script, target)
	cmd.Env = append(os.Environ(),
		"WORKBENCH_GOVERNANCE_TEST_ALLOW_LOCAL_ORIGIN=1",
		"WORKBENCH_GOVERNANCE_TEST_EXPECTED_HEAD="+localHead,
		"WORKBENCH_OPERATION_COMMIT="+desired,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("realign failed: %v\n%s", err, out)
	}
	text := string(out)
	if !strings.Contains(text, "realign=ok") {
		t.Fatalf("missing realign success: %s", text)
	}
	if got := govRealignGitOutput(t, target, "rev-parse", "HEAD"); got != desired {
		t.Fatalf("main head=%s want=%s", got, desired)
	}
	if got := govRealignGitOutput(t, target, "rev-parse", "refs/heads/audit/pre-governance-reset-20260821"); got != localHead {
		t.Fatalf("audit head=%s want=%s", got, localHead)
	}
	if status := govRealignGitOutput(t, target, "status", "--porcelain=v1", "--untracked-files=all"); status != "" {
		t.Fatalf("target remains dirty: %q", status)
	}
}

func TestGovernanceRealignRefusesDirtyCheckout(t *testing.T) {
	base := t.TempDir()
	repo := filepath.Join(base, "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	govRealignGitRun(t, repo, "init", "-b", "main")
	govRealignGitRun(t, repo, "config", "user.email", "test@example.invalid")
	govRealignGitRun(t, repo, "config", "user.name", "Workbench Test")
	govRealignGitRun(t, repo, "remote", "add", "origin", filepath.Join(base, "unused.git"))
	for _, name := range []string{"build.yml", "release.yml", "runner.yml"} {
		govRealignWrite(t, filepath.Join(repo, ".github", "workflows", name), "base\n")
	}
	govRealignGitRun(t, repo, "add", ".")
	govRealignGitRun(t, repo, "commit", "-m", "base")
	for _, name := range []string{"build.yml", "release.yml", "runner.yml"} {
		govRealignWrite(t, filepath.Join(repo, ".github", "workflows", name), "pinned\n")
	}
	govRealignGitRun(t, repo, "add", ".github/workflows")
	govRealignGitRun(t, repo, "commit", "-m", "sec: pin GitHub Actions to full commit SHAs (SEC-008)")
	localHead := govRealignGitOutput(t, repo, "rev-parse", "HEAD")
	govRealignWrite(t, filepath.Join(repo, "unexpected.txt"), "dirty\n")

	script := filepath.Join("..", "..", "scripts", "ops", "governance-realign-workbench-checkout.sh")
	cmd := exec.Command("bash", script, repo)
	cmd.Env = append(os.Environ(),
		"WORKBENCH_GOVERNANCE_TEST_ALLOW_LOCAL_ORIGIN=1",
		"WORKBENCH_GOVERNANCE_TEST_EXPECTED_HEAD="+localHead,
		"WORKBENCH_OPERATION_COMMIT=0123456789abcdef0123456789abcdef01234567",
	)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("dirty checkout unexpectedly realigned: %s", out)
	}
	if !strings.Contains(string(out), "error=checkout-dirty") {
		t.Fatalf("unexpected refusal: %s", out)
	}
}

func govRealignWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func govRealignGitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func govRealignGitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}
