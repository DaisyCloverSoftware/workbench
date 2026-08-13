package core

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublishPreparedChangesetPushesOnlyWorkbenchBranch(t *testing.T) {
	repo := initPrepareTestRepo(t)
	remote := filepath.Join(t.TempDir(), "remote.git")
	prepareTestCommand(t, "git", "clone", "--bare", "--quiet", repo, remote)
	if err := os.WriteFile(filepath.Join(repo, "tracked.txt"), []byte("after\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	prepared, err := PrepareChangeset(context.Background(), repo, "task-publish")
	if err != nil {
		t.Fatal(err)
	}
	activeBefore := prepareTestGit(t, repo, "branch", "--show-current")
	published, err := PublishPreparedChangeset(context.Background(), prepared, remote)
	if err != nil {
		t.Fatal(err)
	}
	if published.Branch != prepared.Branch || published.Commit != prepared.Commit || published.AlreadyPresent {
		t.Fatalf("unexpected publication: %#v", published)
	}
	if got := prepareTestBareGit(t, remote, "rev-parse", "refs/heads/"+prepared.Branch); got != prepared.Commit {
		t.Fatalf("remote branch=%q want %q", got, prepared.Commit)
	}
	if got := prepareTestGit(t, repo, "branch", "--show-current"); got != activeBefore {
		t.Fatalf("publication switched active branch: %q -> %q", activeBefore, got)
	}
	if status := prepareTestGit(t, repo, "status", "--porcelain"); !strings.Contains(status, "tracked.txt") {
		t.Fatalf("publication changed active working tree: %q", status)
	}

	again, err := PublishPreparedChangeset(context.Background(), prepared, remote)
	if err != nil {
		t.Fatal(err)
	}
	if !again.AlreadyPresent || again.Commit != prepared.Commit {
		t.Fatalf("repeat publication was not idempotent: %#v", again)
	}
}

func TestPublishPreparedChangesetRefusesRemoteBranchCollision(t *testing.T) {
	repo := initPrepareTestRepo(t)
	remote := filepath.Join(t.TempDir(), "remote.git")
	prepareTestCommand(t, "git", "clone", "--bare", "--quiet", repo, remote)
	baseline := prepareTestGit(t, repo, "rev-parse", "HEAD")
	if err := os.WriteFile(filepath.Join(repo, "tracked.txt"), []byte("after\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	prepared, err := PrepareChangeset(context.Background(), repo, "task-collision")
	if err != nil {
		t.Fatal(err)
	}
	prepareTestBareGit(t, remote, "update-ref", "refs/heads/"+prepared.Branch, baseline)
	if _, err := PublishPreparedChangeset(context.Background(), prepared, remote); err == nil {
		t.Fatal("expected existing different remote branch to be rejected")
	}
	if got := prepareTestBareGit(t, remote, "rev-parse", "refs/heads/"+prepared.Branch); got != baseline {
		t.Fatalf("collision changed remote ref: got %q want %q", got, baseline)
	}
}

func TestPublishPreparedChangesetRejectsUnsafeRemoteKinds(t *testing.T) {
	for _, remote := range []string{
		"ext::helper example.invalid/repo.git",
		"http://example.invalid/repo.git",
		"https://user@example.invalid/repo.git",
	} {
		if err := validatePublishRemote(remote); err == nil {
			t.Fatalf("expected remote to be rejected: %s", remote)
		}
	}
	for _, remote := range []string{
		"https://example.invalid/repo.git",
		"ssh://git@example.invalid/repo.git",
		"git@example.invalid:org/repo.git",
	} {
		if err := validatePublishRemote(remote); err != nil {
			t.Fatalf("expected remote to be accepted: %s: %v", remote, err)
		}
	}
}

func prepareTestBareGit(t *testing.T, gitDir string, args ...string) string {
	t.Helper()
	cmdArgs := append([]string{"--git-dir", gitDir}, args...)
	cmd := exec.Command("git", cmdArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git --git-dir %s %s: %v\n%s", gitDir, strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

func prepareTestCommand(t *testing.T, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s: %v\n%s", name, strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}
