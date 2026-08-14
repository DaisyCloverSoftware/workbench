package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveRunnerProjectMapsWindowsPathByRepoName(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "workbench")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WORKBENCH_RUNNER_ROOT", root)

	got, err := ResolveRunnerProject(`C:\workspace\workbench`)
	if err != nil {
		t.Fatal(err)
	}
	if got != repo {
		t.Fatalf("got %q want %q", got, repo)
	}
}

func TestResolveRunnerProjectRejectsOutsideRoot(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	t.Setenv("WORKBENCH_RUNNER_ROOT", root)

	if _, err := ResolveRunnerProject(outside); err == nil {
		t.Fatal("expected outside-root path to be rejected")
	}
}

func TestWithinRoot(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "repo")
	if err := os.MkdirAll(inside, 0o755); err != nil {
		t.Fatal(err)
	}
	if !withinRoot(root, inside) {
		t.Fatal("expected path inside root")
	}
	if withinRoot(root, filepath.Dir(root)) {
		t.Fatal("expected parent path to be outside root")
	}
}
