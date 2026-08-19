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
		{Path: "scripts/ops/run.sh", Commit: "deadbeef"},
		{Path: "scripts/ops/run.sh", Commit: strings.Repeat("z", 40)},
		{Path: "scripts/ops/run.sh", Args: []string{"OPENAI_API_KEY=sk-example-abcdefghijklmnopqrstuvwxyz"}},
	} {
		if _, err := validateOperationsScriptRequest(req); err == nil {
			t.Fatalf("unsafe operations script request accepted: %+v", req)
		}
	}
}

func TestOperationsScriptPolicyAcceptsCanonicalCommittedOpsPath(t *testing.T) {
	commit := strings.Repeat("a", 40)
	rel, err := validateOperationsScriptRequest(OperationsScriptRequest{Path: "scripts/ops/deploy-dev.sh", Args: []string{"--dry-run"}, Commit: strings.ToUpper(commit)})
	if err != nil {
		t.Fatal(err)
	}
	if rel != "scripts/ops/deploy-dev.sh" {
		t.Fatalf("rel=%q", rel)
	}
	normalized, err := normalizeOperationsCommit(strings.ToUpper(commit))
	if err != nil || normalized != commit {
		t.Fatalf("normalized commit=%q err=%v", normalized, err)
	}
}

func TestApprovedOperationsOriginIsGitHubOnlyAndCredentialFree(t *testing.T) {
	for _, remote := range []string{
		"https://github.com/DaisyCloverSoftware/workbench.git",
		"https://github.com/DaisyCloverSoftware/workbench",
		"git@github.com:DaisyCloverSoftware/workbench.git",
		"ssh://git@github.com/DaisyCloverSoftware/workbench.git",
	} {
		if !isApprovedOperationsOrigin(remote) {
			t.Fatalf("expected approved github origin: %q", remote)
		}
	}
	for _, remote := range []string{
		"https://token@github.com/DaisyCloverSoftware/workbench.git",
		"https://github.example.com/DaisyCloverSoftware/workbench.git",
		"file:///tmp/workbench.git",
		"/tmp/workbench.git",
		"git@example.com:DaisyCloverSoftware/workbench.git",
		"ssh://matt@github.com/DaisyCloverSoftware/workbench.git",
		"https://github.com/DaisyCloverSoftware/workbench.git?token=oops",
	} {
		if isApprovedOperationsOrigin(remote) {
			t.Fatalf("unsafe operations origin accepted: %q", remote)
		}
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

func TestRunOperationsScriptFetchesExactAdvertisedOriginCommitWithoutChangingCheckout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX bash remote-commit execution smoke")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash unavailable")
	}

	origin := filepath.Join(t.TempDir(), "origin.git")
	gitCommand(t, "", "init", "--bare", origin)
	repo := newOperationsScriptTestRepo(t)
	gitTest(t, repo, "remote", "add", "origin", origin)

	script := filepath.Join(repo, "scripts", "ops", "remote.sh")
	if err := os.MkdirAll(filepath.Dir(script), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "#!/usr/bin/env bash\nset -euo pipefail\nprintf 'source=origin arg=%s commit=%s\\n' \"${1:-none}\" \"${WORKBENCH_OPERATION_COMMIT:-missing}\"\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	gitTest(t, repo, "add", "scripts/ops/remote.sh")
	gitTest(t, repo, "commit", "-m", "publish remote operation")
	remoteCommit := strings.TrimSpace(gitTest(t, repo, "rev-parse", "HEAD"))
	gitTest(t, repo, "push", "origin", "HEAD:refs/heads/main")

	// Put the registered developer checkout back on the older commit so the
	// requested script is absent locally. The executor must fetch into a
	// disposable repository rather than moving this checkout.
	gitTest(t, repo, "reset", "--hard", "HEAD~1")
	before := strings.TrimSpace(gitTest(t, repo, "rev-parse", "HEAD"))
	if _, err := os.Stat(script); !os.IsNotExist(err) {
		t.Fatalf("remote operation should be absent from registered checkout before execution: %v", err)
	}

	oldPredicate := operationsOriginAllowed
	operationsOriginAllowed = func(string) bool { return true }
	t.Cleanup(func() { operationsOriginAllowed = oldPredicate })

	result, err := RunOperationsScript(context.Background(), repo, OperationsScriptRequest{
		Path:   "scripts/ops/remote.sh",
		Args:   []string{"literal;remote"},
		Commit: strings.ToUpper(remoteCommit),
	})
	if err != nil {
		t.Fatal(err)
	}
	after := strings.TrimSpace(gitTest(t, repo, "rev-parse", "HEAD"))
	if before != after {
		t.Fatalf("remote operation moved registered checkout: before=%s after=%s", before, after)
	}
	if _, err := os.Stat(script); !os.IsNotExist(err) {
		t.Fatalf("remote operation materialised into registered checkout: %v", err)
	}
	if result.Commit != remoteCommit || len(result.ScriptSHA256) != 64 || result.Transport != "github-origin-commit-worktree-bash" || result.ExitCode != 0 {
		t.Fatalf("unexpected remote result metadata: %+v", result)
	}
	if !strings.Contains(result.Output, "source=origin") || !strings.Contains(result.Output, "arg=literal;remote") || !strings.Contains(result.Output, "commit="+remoteCommit) {
		t.Fatalf("unexpected remote operation output: %q", result.Output)
	}
}

func TestRunOperationsScriptRejectsCommitThatIsNotAdvertisedByOriginHead(t *testing.T) {
	origin := filepath.Join(t.TempDir(), "origin.git")
	gitCommand(t, "", "init", "--bare", origin)
	repo := newOperationsScriptTestRepo(t)
	gitTest(t, repo, "remote", "add", "origin", origin)

	script := filepath.Join(repo, "scripts", "ops", "old.sh")
	if err := os.MkdirAll(filepath.Dir(script), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho old\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	gitTest(t, repo, "add", "scripts/ops/old.sh")
	gitTest(t, repo, "commit", "-m", "old operation")
	oldCommit := strings.TrimSpace(gitTest(t, repo, "rev-parse", "HEAD"))
	gitTest(t, repo, "push", "origin", "HEAD:refs/heads/main")

	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("new head\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitTest(t, repo, "add", "README.md")
	gitTest(t, repo, "commit", "-m", "move branch head")
	gitTest(t, repo, "push", "origin", "HEAD:refs/heads/main")

	oldPredicate := operationsOriginAllowed
	operationsOriginAllowed = func(string) bool { return true }
	t.Cleanup(func() { operationsOriginAllowed = oldPredicate })

	if _, err := RunOperationsScript(context.Background(), repo, OperationsScriptRequest{Path: "scripts/ops/old.sh", Commit: oldCommit}); err == nil || !strings.Contains(err.Error(), "not currently advertised") {
		t.Fatalf("stale origin commit should be refused, got %v", err)
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
	return gitCommand(t, repo, args...)
}

func gitCommand(t *testing.T, repo string, args ...string) string {
	t.Helper()
	commandArgs := append([]string(nil), args...)
	if repo != "" {
		commandArgs = append([]string{"-C", repo}, commandArgs...)
	}
	cmd := exec.Command("git", commandArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", commandArgs, err, out)
	}
	return string(out)
}
