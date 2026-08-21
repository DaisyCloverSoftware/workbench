//go:build !windows

package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGovernanceCleanupScriptCleansOnlyAuditedRelayExperiment(t *testing.T) {
	repo := newGovernanceCleanupFixture(t)
	writeFile(t, filepath.Join(repo, "internal/core/relay_state.go"), "dirty relay state\n")
	writeFile(t, filepath.Join(repo, "internal/core/relay_lock.go"), "package core\n")
	writeFile(t, filepath.Join(repo, "internal/core/relay_lock_test.go"), "package core\n")
	writeFile(t, filepath.Join(repo, "internal/core/relay_state_concurrency_test.go"), "package core\n")

	script := filepath.Join("..", "..", "scripts", "ops", "governance-clean-workbench-checkout.sh")
	cmd := exec.Command("bash", script, repo)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cleanup failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "cleanup=ok") {
		t.Fatalf("cleanup output missing success marker: %s", out)
	}
	status := gitOutput(t, repo, "status", "--porcelain=v1", "--untracked-files=all")
	if strings.TrimSpace(status) != "" {
		t.Fatalf("checkout remains dirty: %q", status)
	}
	body, err := os.ReadFile(filepath.Join(repo, "internal/core/relay_state.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "canonical relay state\n" {
		t.Fatalf("tracked file was not restored to HEAD: %q", body)
	}
}

func TestGovernanceCleanupScriptRefusesUnexpectedDirtyPath(t *testing.T) {
	repo := newGovernanceCleanupFixture(t)
	writeFile(t, filepath.Join(repo, "internal/core/relay_state.go"), "dirty relay state\n")
	writeFile(t, filepath.Join(repo, "internal/core/relay_lock.go"), "package core\n")
	writeFile(t, filepath.Join(repo, "internal/core/relay_lock_test.go"), "package core\n")
	writeFile(t, filepath.Join(repo, "internal/core/relay_state_concurrency_test.go"), "package core\n")
	writeFile(t, filepath.Join(repo, "unexpected.txt"), "must survive\n")

	script := filepath.Join("..", "..", "scripts", "ops", "governance-clean-workbench-checkout.sh")
	cmd := exec.Command("bash", script, repo)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("cleanup unexpectedly accepted extra dirty path: %s", out)
	}
	if !strings.Contains(string(out), "error=unexpected-untracked-set") {
		t.Fatalf("unexpected refusal output: %s", out)
	}
	if _, err := os.Stat(filepath.Join(repo, "unexpected.txt")); err != nil {
		t.Fatalf("refused cleanup modified unexpected file: %v", err)
	}
}

func newGovernanceCleanupFixture(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	gitRun(t, repo, "init", "-b", "main")
	gitRun(t, repo, "config", "user.email", "test@example.invalid")
	gitRun(t, repo, "config", "user.name", "Workbench Test")
	gitRun(t, repo, "remote", "add", "origin", "https://github.com/DaisyCloverSoftware/workbench.git")
	writeFile(t, filepath.Join(repo, "internal/core/relay_state.go"), "canonical relay state\n")
	gitRun(t, repo, "add", "internal/core/relay_state.go")
	gitRun(t, repo, "commit", "-m", "fixture")
	return repo
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
	return string(out)
}
