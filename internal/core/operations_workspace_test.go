package core

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestOperationsWorkspaceAllowsDirtySourceWithoutCopyingLocalEdits(t *testing.T) {
	repo := initOperationsWorkspaceRepo(t)
	readme := filepath.Join(repo, "README.md")
	if err := os.WriteFile(readme, []byte("local uncommitted edit\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	ws, err := CreateOperationsTaskWorkspace(ctx, repo, "task-dirty-operations")
	if err != nil {
		t.Fatalf("operations workspace should not be blocked by dirty source: %v", err)
	}
	defer func() { _ = RemoveTaskWorkspace(ctx, repo, ws.TaskID) }()

	workspaceBody, err := os.ReadFile(filepath.Join(ws.Workspace, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	workspaceText := strings.ReplaceAll(string(workspaceBody), "\r\n", "\n")
	if workspaceText != "committed state\n" {
		t.Fatalf("operations workspace copied uncommitted source edits: %q", workspaceBody)
	}
	sourceBody, err := os.ReadFile(readme)
	if err != nil {
		t.Fatal(err)
	}
	if string(sourceBody) != "local uncommitted edit\n" {
		t.Fatalf("operations workspace altered user's source checkout: %q", sourceBody)
	}
	workspaceChanges, err := InspectChangeset(ctx, ws.Workspace)
	if err != nil {
		t.Fatal(err)
	}
	if !workspaceChanges.Clean {
		t.Fatalf("fresh operations workspace should be clean: %+v", workspaceChanges)
	}
	sourceChanges, err := InspectChangeset(ctx, repo)
	if err != nil {
		t.Fatal(err)
	}
	if sourceChanges.Clean {
		t.Fatal("source checkout unexpectedly lost its uncommitted edit")
	}
}

func TestCodingWorkspaceStillRejectsDirtySource(t *testing.T) {
	repo := initOperationsWorkspaceRepo(t)
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("local edit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := CreateTaskWorkspace(context.Background(), repo, "task-coding-dirty")
	if err == nil || !strings.Contains(err.Error(), "source worktree has local changes") {
		t.Fatalf("coding workspace must continue rejecting dirty source: %v", err)
	}
}

func initOperationsWorkspaceRepo(t *testing.T) string {
	t.Helper()
	repo := filepath.Join(t.TempDir(), "repo")
	if out, err := exec.Command("git", "init", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	for _, args := range [][]string{{"config", "user.email", "workbench-test@example.invalid"}, {"config", "user.name", "Workbench Test"}} {
		if out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git config: %v: %s", err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("committed state\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", repo, "add", "README.md").CombinedOutput(); err != nil {
		t.Fatalf("git add: %v: %s", err, out)
	}
	if out, err := exec.Command("git", "-C", repo, "commit", "-m", "fixture").CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v: %s", err, out)
	}
	return repo
}
