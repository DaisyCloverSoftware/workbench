package core

import (
	"os"
	"path/filepath"
	"runtime"
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
	gotInfo, err := os.Stat(got)
	if err != nil {
		t.Fatal(err)
	}
	repoInfo, err := os.Stat(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(gotInfo, repoInfo) {
		t.Fatalf("got %q want repository %q", got, repo)
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

func TestResolveRunnerProjectRejectsSymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation is not guaranteed for unprivileged Windows CI")
	}
	root := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(root, "escape")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WORKBENCH_RUNNER_ROOT", root)

	if _, err := ResolveRunnerProject(`C:\workspace\escape`); err == nil {
		t.Fatal("expected in-root symlink to outside directory to be rejected")
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
