//go:build !windows

package core

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRum160WorkspaceDiscovery(t *testing.T) {
	for _, program := range []string{"bash", "git", "timeout"} {
		if _, err := exec.LookPath(program); err != nil {
			t.Skip(program + " is required")
		}
	}
	script, err := filepath.Abs("../../scripts/ops/rum160-readonly-workspace-discovery.sh")
	if err != nil {
		t.Fatal(err)
	}
	git := func(dir string, args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("fixture git %v: %v: %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	root := filepath.Join(t.TempDir(), "projects")
	repo := filepath.Join(root, "rum")
	if err := os.MkdirAll(repo, 0700); err != nil {
		t.Fatal(err)
	}
	git(repo, "init", "-q", "--initial-branch=main")
	git(repo, "config", "user.name", "Diagnostic Fixture")
	git(repo, "config", "user.email", "fixture@example.invalid")
	git(repo, "remote", "add", "origin", "https://github.com/DaisyCloverSoftware/rum.git")
	put := func(path, text string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	put(filepath.Join(repo, "base.txt"), "base\n")
	git(repo, "add", "base.txt")
	git(repo, "commit", "-qm", "fixture base")
	head := git(repo, "rev-parse", "HEAD")
	run := func(config string, args ...string) (string, error) {
		t.Helper()
		cmd := exec.Command("bash", append([]string{script}, args...)...)
		cmd.Dir = repo
		var env []string
		for _, entry := range os.Environ() {
			if !strings.HasPrefix(entry, "WORKBENCH_RUNNER_ROOT") && !strings.HasPrefix(entry, "GIT_") {
				env = append(env, entry)
			}
		}
		cmd.Env = append(env, "WORKBENCH_RUNNER_ROOTS="+config)
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	t.Run("reject-arguments", func(t *testing.T) {
		for _, args := range [][]string{nil, {"main"}, {head, "/"}} {
			out, err := run(root, args...)
			if err == nil || !strings.Contains(out, "error=exact-pr-head-required") {
				t.Fatalf("accepted %v: %s", args, out)
			}
		}
	})
	t.Run("missing-explicit-root", func(t *testing.T) {
		out, err := run(filepath.Join(root, "missing"), head)
		if err == nil || !strings.Contains(out, "error=configured-root-unavailable") {
			t.Fatal(out, err)
		}
	})
	t.Run("root-limit", func(t *testing.T) {
		out, err := run(strings.Repeat(root+":", 8)+root, head)
		if err == nil || !strings.Contains(out, "error=root-limit") {
			t.Fatal(out, err)
		}
	})
	t.Run("reject-other-origin", func(t *testing.T) {
		git(repo, "remote", "set-url", "origin", "https://github.com/example/other.git")
		defer git(repo, "remote", "set-url", "origin", "https://github.com/DaisyCloverSoftware/rum.git")
		out, err := run(root, head)
		if err != nil || !strings.Contains(out, "candidate_wrong_origin=") || !strings.Contains(out, "verified_worktrees=0") {
			t.Fatal(out, err)
		}
	})
	t.Run("reject-symlink-alias", func(t *testing.T) {
		aliasRoot := t.TempDir()
		if err := os.Symlink(repo, filepath.Join(aliasRoot, "rum")); err != nil {
			t.Fatal(err)
		}
		out, err := run(aliasRoot, head)
		if err != nil || !strings.Contains(out, "candidate_unavailable=") || !strings.Contains(out, "verified_worktrees=0") {
			t.Fatal(out, err)
		}
	})
	t.Run("dirty-linked-exact-head-readonly", func(t *testing.T) {
		put(filepath.Join(repo, "local.txt"), "local-only\n")
		git(repo, "add", "local.txt")
		git(repo, "commit", "-qm", "fixture local-only")
		git(repo, "update-ref", "refs/remotes/origin/fixture-pr", "HEAD")
		linked := filepath.Join(t.TempDir(), "linked rum")
		git(repo, "worktree", "add", "--detach", linked, head)
		put(filepath.Join(repo, "staged.txt"), "staged\n")
		git(repo, "add", "staged.txt")
		put(filepath.Join(repo, "base.txt"), "unstaged\n")
		put(filepath.Join(repo, "recovered.patch"), "fixture artifact\n")
		indexPath := filepath.Join(repo, ".git", "index")
		indexBefore, err := os.ReadFile(indexPath)
		if err != nil {
			t.Fatal(err)
		}
		monitor := filepath.Join(t.TempDir(), "fsmonitor")
		marker := monitor + "-executed"
		if err := os.WriteFile(monitor, []byte("#!/bin/sh\ntouch '"+marker+"'\n"), 0700); err != nil {
			t.Fatal(err)
		}
		git(repo, "config", "core.fsmonitor", monitor)
		out, err := run(root, head)
		if err != nil {
			t.Fatal(out, err)
		}
		for _, expected := range []string{"verified_worktrees=2", "_commits_not_in_pr_total=1", "_commits_not_in_pr_total=0", "_status_total=3", "_staged_entries=1", "_unstaged_entries=1", "_untracked_entries=1", "workspace_ready=false", "task_ownership=not_assessed"} {
			if !strings.Contains(out, expected) {
				t.Fatalf("missing %s in %s", expected, out)
			}
		}
		indexAfter, err := os.ReadFile(indexPath)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(indexBefore, indexAfter) {
			t.Fatal("diagnostic changed index bytes")
		}
		for file, expected := range map[string]string{"base.txt": "unstaged\n", "staged.txt": "staged\n", "recovered.patch": "fixture artifact\n"} {
			b, err := os.ReadFile(filepath.Join(repo, file))
			if err != nil || string(b) != expected {
				t.Fatalf("changed %s", file)
			}
		}
		if _, err := os.Stat(marker); !os.IsNotExist(err) {
			t.Fatal("repository fsmonitor executed")
		}
	})
}
