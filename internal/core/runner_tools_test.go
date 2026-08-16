package core

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRunnerToolListsOnlyRepositoryRoots(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WORKBENCH_RUNNER_ROOT", root)
	repo := filepath.Join(root, "garage")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "init", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	if err := os.MkdirAll(filepath.Join(root, "not-a-repo"), 0o755); err != nil {
		t.Fatal(err)
	}

	response, err := ApplyRunnerToolRequest(context.Background(), RunnerToolRequest{Action: "list_projects"})
	if err != nil {
		t.Fatal(err)
	}
	if !response.OK || len(response.Projects) != 1 {
		t.Fatalf("unexpected response: %+v", response)
	}
	if response.Projects[0].Name != "garage" || response.Projects[0].Ref != "runner://garage" {
		t.Fatalf("unexpected project: %+v", response.Projects[0])
	}
}
