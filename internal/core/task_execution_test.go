package core

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestFinalizeTaskWorkspacePreparesReviewWithoutDirtyingSource(t *testing.T) {
	isolateKnowledgeConfig(t)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	repo := initPrepareTestRepo(t)
	ctx := context.Background()
	ws, err := CreateTaskWorkspace(ctx, repo, "task-finalize")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws.Workspace, "tracked.txt"), []byte("worker change\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := FinalizeTaskWorkspace(ctx, ws)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed || result.Published || result.Branch == "" || result.Commit == "" {
		t.Fatalf("unexpected review result: %#v", result)
	}
	if status := prepareTestGit(t, repo, "status", "--porcelain"); status != "" {
		t.Fatalf("source checkout was dirtied by isolated finalization: %q", status)
	}
	if got := prepareTestGit(t, repo, "show", result.Commit+":tracked.txt"); got != "worker change" {
		t.Fatalf("prepared review content=%q", got)
	}
	if got := prepareTestGit(t, repo, "rev-parse", result.Branch); got != result.Commit {
		t.Fatalf("review branch=%q commit=%q", got, result.Commit)
	}
	if _, ok, err := OpenTaskWorkspace(repo, "task-finalize"); err != nil || ok {
		t.Fatalf("successful workspace was not cleaned up: ok=%t err=%v", ok, err)
	}
}

func TestFinalizeTaskWorkspacePublishesThroughPrivatePolicy(t *testing.T) {
	isolateKnowledgeConfig(t)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	repo := initPrepareTestRepo(t)
	remote := filepath.Join(t.TempDir(), "review.git")
	cmd := exec.Command("git", "init", "--bare", "-q", remote)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %v\n%s", err, out)
	}
	if _, err := SavePublicationPolicy(PublicationPolicy{Project: repo, Mode: PublicationPublish, RemoteURL: remote}); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	ws, err := CreateTaskWorkspace(ctx, repo, "task-publish")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws.Workspace, "tracked.txt"), []byte("published change\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := FinalizeTaskWorkspace(ctx, ws)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed || !result.Published || result.Branch == "" || result.Commit == "" {
		t.Fatalf("unexpected published review result: %#v", result)
	}
	if status := prepareTestGit(t, repo, "status", "--porcelain"); status != "" {
		t.Fatalf("source checkout was dirtied by publication: %q", status)
	}
	out, err := exec.Command("git", "--git-dir", remote, "rev-parse", "refs/heads/"+result.Branch).CombinedOutput()
	if err != nil {
		t.Fatalf("read published branch: %v\n%s", err, out)
	}
	if strings.TrimSpace(string(out)) != result.Commit {
		t.Fatalf("published branch=%q commit=%q", strings.TrimSpace(string(out)), result.Commit)
	}
}

func TestFinalizeTaskWorkspaceRefusesWorkerCommit(t *testing.T) {
	isolateKnowledgeConfig(t)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	repo := initPrepareTestRepo(t)
	ctx := context.Background()
	ws, err := CreateTaskWorkspace(ctx, repo, "task-commit-refusal")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws.Workspace, "tracked.txt"), []byte("committed by worker\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	prepareTestGit(t, ws.Workspace, "add", "tracked.txt")
	prepareTestGit(t, ws.Workspace, "-c", "user.name=Worker", "-c", "user.email=worker@example.invalid", "commit", "-q", "-m", "worker commit")

	if _, err := FinalizeTaskWorkspace(ctx, ws); err == nil || !strings.Contains(err.Error(), "worker created a commit") {
		t.Fatalf("expected worker commit refusal, got %v", err)
	}
	if status := prepareTestGit(t, repo, "status", "--porcelain"); status != "" {
		t.Fatalf("source changed after worker commit refusal: %q", status)
	}
}
