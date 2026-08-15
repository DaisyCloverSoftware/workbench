package core

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTaskWorkspaceIsIsolatedAndRecoverable(t *testing.T) {
	isolateKnowledgeConfig(t)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	repo := initPrepareTestRepo(t)
	ctx := context.Background()
	baseBranch := prepareTestGit(t, repo, "branch", "--show-current")

	ws, err := CreateTaskWorkspace(ctx, repo, "task-one")
	if err != nil {
		t.Fatal(err)
	}
	if ws.Workspace == repo || ws.BaseRevision == "" || ws.TaskID != "task-one" {
		t.Fatalf("unexpected workspace: %#v", ws)
	}
	if ws.BaseBranch != baseBranch {
		t.Fatalf("workspace base branch=%q want %q", ws.BaseBranch, baseBranch)
	}
	if got := prepareTestGit(t, ws.Workspace, "rev-parse", "HEAD"); got != ws.BaseRevision {
		t.Fatalf("workspace baseline=%s want %s", got, ws.BaseRevision)
	}
	if got := prepareTestGit(t, ws.Workspace, "branch", "--show-current"); got != "" {
		t.Fatalf("workspace should be detached, got branch %q", got)
	}
	if err := os.WriteFile(filepath.Join(ws.Workspace, "tracked.txt"), []byte("task edit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := prepareTestGit(t, repo, "show", "HEAD:tracked.txt"); got != "before" {
		t.Fatalf("source repository changed: %q", got)
	}
	if status := prepareTestGit(t, repo, "status", "--porcelain"); status != "" {
		t.Fatalf("source repository became dirty: %q", status)
	}

	opened, ok, err := OpenTaskWorkspace(repo, "task-one")
	if err != nil || !ok {
		t.Fatalf("open workspace: ok=%t err=%v", ok, err)
	}
	if opened.Workspace != ws.Workspace || opened.BaseRevision != ws.BaseRevision || opened.BaseBranch != ws.BaseBranch {
		t.Fatalf("unexpected reopened workspace: %#v", opened)
	}
	if _, err := CreateTaskWorkspace(ctx, repo, "task-one"); err != nil {
		t.Fatalf("idempotent create failed: %v", err)
	}

	if err := RemoveTaskWorkspace(ctx, repo, "task-one"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(ws.Workspace); !os.IsNotExist(err) {
		t.Fatalf("workspace still exists after cleanup: %v", err)
	}
	if _, ok, err := OpenTaskWorkspace(repo, "task-one"); err != nil || ok {
		t.Fatalf("workspace metadata remained after cleanup: ok=%t err=%v", ok, err)
	}
}

func TestTaskWorkspaceAcceptsLegacyV1Metadata(t *testing.T) {
	isolateKnowledgeConfig(t)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	repo := initPrepareTestRepo(t)
	ctx := context.Background()
	ws, err := CreateTaskWorkspace(ctx, repo, "task-legacy")
	if err != nil {
		t.Fatal(err)
	}
	_, metadataPath, err := taskWorkspacePaths(ws.Project, "task-legacy")
	if err != nil {
		t.Fatal(err)
	}
	ws.Version = 1
	ws.BaseBranch = ""
	if err := saveTaskWorkspace(metadataPath, ws); err != nil {
		t.Fatal(err)
	}
	opened, ok, err := OpenTaskWorkspace(repo, "task-legacy")
	if err != nil || !ok {
		t.Fatalf("legacy workspace should remain recoverable: ok=%t err=%v", ok, err)
	}
	if opened.Version != 1 || opened.BaseBranch != "" {
		t.Fatalf("legacy metadata changed unexpectedly: %#v", opened)
	}
	if err := RemoveTaskWorkspace(ctx, repo, "task-legacy"); err != nil {
		t.Fatal(err)
	}
}

func TestTaskWorkspaceRequiresCleanRepository(t *testing.T) {
	isolateKnowledgeConfig(t)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	repo := initPrepareTestRepo(t)
	if err := os.WriteFile(filepath.Join(repo, "tracked.txt"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateTaskWorkspace(context.Background(), repo, "task-dirty"); err == nil || !strings.Contains(err.Error(), "clean") {
		t.Fatalf("expected clean-repository refusal, got %v", err)
	}
}

func TestTaskWorkspaceRequiresRepositoryRoot(t *testing.T) {
	isolateKnowledgeConfig(t)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	repo := initPrepareTestRepo(t)
	sub := filepath.Join(repo, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateTaskWorkspace(context.Background(), sub, "task-subdir"); err == nil {
		t.Fatal("expected repository-root boundary")
	}
}
