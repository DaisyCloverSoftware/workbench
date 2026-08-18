package core

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func initCommittedRunnerRepo(t *testing.T, root, name string) string {
	t.Helper()
	repo := filepath.Join(root, name)
	if out, err := exec.Command("git", "init", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	for _, args := range [][]string{{"config", "user.email", "workbench-test@example.invalid"}, {"config", "user.name", "Workbench Test"}} {
		if out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git config: %v: %s", err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("fixture\n"), 0o644); err != nil {
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

func TestRunnerProjectDiscoveryHidesLinkedWorktreesAndBootstrapBackups(t *testing.T) {
	root := t.TempDir()
	canonical := initCommittedRunnerRepo(t, root, "workbench")
	worktree := filepath.Join(root, "workbench-fix-review")
	if out, err := exec.Command("git", "-C", canonical, "worktree", "add", "-b", "test-review", worktree).CombinedOutput(); err != nil {
		t.Fatalf("git worktree add: %v: %s", err, out)
	}
	initCommittedRunnerRepo(t, root, "workbench-pre-bootstrap-20260818-120000")
	useRunnerRoots(t, root)

	response, err := ApplyRunnerToolRequest(context.Background(), RunnerToolRequest{Action: "list_projects"})
	if err != nil {
		t.Fatal(err)
	}
	if !response.OK {
		t.Fatalf("list projects failed: %+v", response)
	}
	if len(response.Projects) != 1 {
		t.Fatalf("projects=%+v want only canonical checkout", response.Projects)
	}
	if response.Projects[0].Name != "workbench" || response.Projects[0].Ref != "runner://workbench" {
		t.Fatalf("unexpected canonical project: %+v", response.Projects[0])
	}
}
