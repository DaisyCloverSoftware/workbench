package core

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTransferTaskWorkspaceChangesMatchesIsolatedFingerprint(t *testing.T) {
	isolateKnowledgeConfig(t)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	repo := initPrepareTestRepo(t)
	ctx := context.Background()
	ws, err := CreateTaskWorkspace(ctx, repo, "task-transfer")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws.Workspace, "tracked.txt"), []byte("after\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws.Workspace, "added.txt"), []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	workspaceSnapshot, err := SnapshotChangeset(ctx, ws.Workspace)
	if err != nil {
		t.Fatal(err)
	}

	result, err := TransferTaskWorkspaceChanges(ctx, ws)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed || result.Prepared.Fingerprint != workspaceSnapshot.Fingerprint {
		t.Fatalf("unexpected transfer result: %#v", result)
	}
	if got := prepareTestGit(t, repo, "status", "--porcelain"); !strings.Contains(got, "tracked.txt") || !strings.Contains(got, "added.txt") {
		t.Fatalf("transferred source changes missing: %q", got)
	}
	if got, err := os.ReadFile(filepath.Join(repo, "tracked.txt")); err != nil || string(got) != "after\n" {
		t.Fatalf("tracked content=%q err=%v", got, err)
	}
	if got, err := os.ReadFile(filepath.Join(repo, "added.txt")); err != nil || string(got) != "new\n" {
		t.Fatalf("added content=%q err=%v", got, err)
	}
}

func TestTransferTaskWorkspaceChangesRefusesConcurrentSourceEdit(t *testing.T) {
	isolateKnowledgeConfig(t)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	repo := initPrepareTestRepo(t)
	ctx := context.Background()
	ws, err := CreateTaskWorkspace(ctx, repo, "task-source-race")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws.Workspace, "tracked.txt"), []byte("worker\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "tracked.txt"), []byte("human\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := TransferTaskWorkspaceChanges(ctx, ws); err == nil || !strings.Contains(err.Error(), "source worktree changed") {
		t.Fatalf("expected concurrent source edit refusal, got %v", err)
	}
	if got, _ := os.ReadFile(filepath.Join(repo, "tracked.txt")); string(got) != "human\n" {
		t.Fatalf("source edit was overwritten: %q", got)
	}
}

func TestTransferTaskWorkspaceChangesRefusesWorkerCommit(t *testing.T) {
	isolateKnowledgeConfig(t)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	repo := initPrepareTestRepo(t)
	ctx := context.Background()
	ws, err := CreateTaskWorkspace(ctx, repo, "task-worker-commit")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws.Workspace, "tracked.txt"), []byte("committed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	prepareTestGit(t, ws.Workspace, "add", "tracked.txt")
	prepareTestGit(t, ws.Workspace, "-c", "user.name=Worker", "-c", "user.email=worker@example.invalid", "commit", "-q", "-m", "worker commit")
	if _, err := TransferTaskWorkspaceChanges(ctx, ws); err == nil || !strings.Contains(err.Error(), "worker created a commit") {
		t.Fatalf("expected worker commit refusal, got %v", err)
	}
	if status := prepareTestGit(t, repo, "status", "--porcelain"); status != "" {
		t.Fatalf("source repository changed after worker commit refusal: %q", status)
	}
}
