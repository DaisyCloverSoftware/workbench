package core

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestOperationsScriptPolicyRejectsArbitraryShellPathsAndSecrets(t *testing.T) {
	for _, req := range []OperationsScriptRequest{
		{Path: "deploy.sh"},
		{Path: "scripts/deploy/run.sh"},
		{Path: "scripts/ops/../deploy.sh"},
		{Path: "/tmp/run.sh"},
		{Path: "scripts/ops/run.py"},
		{Path: "scripts/ops/run.sh", Args: []string{"OPENAI_API_KEY=sk-example-abcdefghijklmnopqrstuvwxyz"}},
	} {
		if _, err := validateOperationsScriptRequest(req); err == nil {
			t.Fatalf("unsafe operations script request accepted: %+v", req)
		}
	}
}

func TestOperationsScriptPolicyAcceptsCanonicalCommittedOpsPath(t *testing.T) {
	rel, err := validateOperationsScriptRequest(OperationsScriptRequest{Path: "scripts/ops/deploy-dev.sh", Args: []string{"--dry-run"}})
	if err != nil {
		t.Fatal(err)
	}
	if rel != "scripts/ops/deploy-dev.sh" {
		t.Fatalf("rel=%q", rel)
	}
}

func TestRunOperationsScriptUsesCommittedDetachedWorktree(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX bash worktree execution smoke")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash unavailable")
	}
	repo := newOperationsScriptTestRepo(t)
	script := filepath.Join(repo, "scripts", "ops", "demo.sh")
	if err := os.MkdirAll(filepath.Dir(script), 0o755); err != nil {
		t.Fatal(err)
	}
	committed := "#!/usr/bin/env bash\nset -euo pipefail\nprintf 'source=committed arg=%s commit=%s\\n' \"${1:-none}\" \"${WORKBENCH_OPERATION_COMMIT:-missing}\"\n"
	if err := os.WriteFile(script, []byte(committed), 0o755); err != nil {
		t.Fatal(err)
	}
	gitTest(t, repo, "add", "scripts/ops/demo.sh")
	gitTest(t, repo, "commit", "-m", "add operation")
	head := strings.TrimSpace(gitTest(t, repo, "rev-parse", "HEAD"))

	// Dirty the developer checkout after the commit. The executor must ignore
	// this and run the exact committed script from its detached worktree.
	if err := os.WriteFile(script, []byte("#!/usr/bin/env bash\necho source=DIRTY\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := RunOperationsScript(context.Background(), repo, OperationsScriptRequest{
		Path: "scripts/ops/demo.sh",
		Args: []string{"literal;not-shell"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Commit != head || len(result.ScriptSHA256) != 64 || result.Transport != "git-worktree-bash" || result.ExitCode != 0 {
		t.Fatalf("unexpected result metadata: %+v", result)
	}
	if !strings.Contains(result.Output, "source=committed") || strings.Contains(result.Output, "DIRTY") {
		t.Fatalf("dirty checkout affected operation: %q", result.Output)
	}
	if !strings.Contains(result.Output, "arg=literal;not-shell") {
		t.Fatalf("literal argv not preserved: %q", result.Output)
	}
	if !strings.Contains(result.Output, "commit="+head) {
		t.Fatalf("commit environment missing: %q", result.Output)
	}
}

func TestRunOperationsScriptRejectsUntrackedAndSymlinkOpsFiles(t *testing.T) {
	repo := newOperationsScriptTestRepo(t)
	untracked := filepath.Join(repo, "scripts", "ops", "untracked.sh")
	if err := os.MkdirAll(filepath.Dir(untracked), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(untracked, []byte("#!/bin/sh\necho nope\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := RunOperationsScript(context.Background(), repo, OperationsScriptRequest{Path: "scripts/ops/untracked.sh"}); err == nil || !strings.Contains(err.Error(), "not tracked") {
		t.Fatalf("untracked script should be refused, got %v", err)
	}

	if runtime.GOOS == "windows" {
		return
	}
	target := filepath.Join(repo, "target.sh")
	if err := os.WriteFile(target, []byte("#!/bin/sh\necho target\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(repo, "scripts", "ops", "link.sh")
	if err := os.Symlink("../../target.sh", link); err != nil {
		t.Fatal(err)
	}
	gitTest(t, repo, "add", "target.sh", "scripts/ops/link.sh")
	gitTest(t, repo, "commit", "-m", "add symlink")
	if _, err := RunOperationsScript(context.Background(), repo, OperationsScriptRequest{Path: "scripts/ops/link.sh"}); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("symlink operations script should be refused, got %v", err)
	}
}

func newOperationsScriptTestRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	gitTest(t, repo, "init")
	gitTest(t, repo, "config", "user.email", "workbench-test@example.invalid")
	gitTest(t, repo, "config", "user.name", "Workbench Test")
	readme := filepath.Join(repo, "README.md")
	if err := os.WriteFile(readme, []byte("test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitTest(t, repo, "add", "README.md")
	gitTest(t, repo, "commit", "-m", "initial")
	return repo
}

func gitTest(t *testing.T, repo string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return string(out)
}
