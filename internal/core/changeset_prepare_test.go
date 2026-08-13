package core

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrepareChangesetCreatesIsolatedLocalBranch(t *testing.T) {
	repo := initPrepareTestRepo(t)
	if err := os.WriteFile(filepath.Join(repo, "tracked.txt"), []byte("after\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "added.txt"), []byte("added\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	beforeBranch := prepareTestGit(t, repo, "branch", "--show-current")
	before, err := SnapshotChangeset(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := PrepareChangeset(context.Background(), repo, "task-123")
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Fingerprint != before.Fingerprint || prepared.BaseRevision != before.Inspection.BaseRevision {
		t.Fatalf("prepared snapshot changed identity: %#v vs %#v", prepared, before)
	}
	if !strings.HasPrefix(prepared.Branch, "workbench/task-123-") {
		t.Fatalf("unexpected branch: %s", prepared.Branch)
	}
	if got := prepareTestGit(t, repo, "branch", "--show-current"); got != beforeBranch {
		t.Fatalf("active branch changed: %q -> %q", beforeBranch, got)
	}
	status := prepareTestGit(t, repo, "status", "--porcelain")
	if !strings.Contains(status, "tracked.txt") || !strings.Contains(status, "added.txt") {
		t.Fatalf("active working changes were not preserved: %q", status)
	}
	if got := prepareTestGit(t, repo, "show", prepared.Commit+":tracked.txt"); got != "after" {
		t.Fatalf("prepared tracked content=%q", got)
	}
	if got := prepareTestGit(t, repo, "show", prepared.Commit+":added.txt"); got != "added" {
		t.Fatalf("prepared added content=%q", got)
	}
	if got := prepareTestGit(t, repo, "rev-parse", prepared.Branch); got != prepared.Commit {
		t.Fatalf("branch tip=%q commit=%q", got, prepared.Commit)
	}

	again, err := PrepareChangeset(context.Background(), repo, "task-123")
	if err != nil {
		t.Fatal(err)
	}
	if again.Branch != prepared.Branch || again.Commit != prepared.Commit {
		t.Fatalf("preparation was not idempotent: first=%#v again=%#v", prepared, again)
	}
}

func TestPrepareChangesetRejectsProtectedPath(t *testing.T) {
	repo := initPrepareTestRepo(t)
	if err := os.WriteFile(filepath.Join(repo, ".env"), []byte("mode=test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := PrepareChangeset(context.Background(), repo, "task-protected"); err == nil {
		t.Fatal("expected protected path to be rejected")
	}
}

func TestPreparedBranchNameDoesNotUseFreeformText(t *testing.T) {
	got := preparedBranchName("Task 42: fix thing!", strings.Repeat("a", 64))
	if got != "workbench/task-42-fix-thing-aaaaaaaaaaaa" {
		t.Fatalf("unexpected branch name: %s", got)
	}
}

func initPrepareTestRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	prepareTestGit(t, repo, "init", "-q")
	prepareTestGit(t, repo, "config", "user.name", "Workbench Test")
	prepareTestGit(t, repo, "config", "user.email", "workbench-test@example.invalid")
	prepareTestGit(t, repo, "config", "core.autocrlf", "false")
	if err := os.WriteFile(filepath.Join(repo, "tracked.txt"), []byte("before\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	prepareTestGit(t, repo, "add", "tracked.txt")
	prepareTestGit(t, repo, "commit", "-q", "-m", "baseline")
	return repo
}

func prepareTestGit(t *testing.T, repo string, args ...string) string {
	t.Helper()
	cmdArgs := append([]string{"-C", repo}, args...)
	cmd := exec.Command("git", cmdArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}
