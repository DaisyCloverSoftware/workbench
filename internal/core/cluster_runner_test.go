package core

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func useSingleRunnerRoot(t *testing.T, root string) {
	t.Helper()
	t.Setenv("WORKBENCH_RUNNER_ROOTS", "")
	t.Setenv("WORKBENCH_RUNNER_ROOT", root)
}

func useRunnerRoots(t *testing.T, roots ...string) {
	t.Helper()
	t.Setenv("WORKBENCH_RUNNER_ROOT", "")
	t.Setenv("WORKBENCH_RUNNER_ROOTS", strings.Join(roots, string(os.PathListSeparator)))
}

func TestResolveRunnerProjectMapsWindowsPathByRepoName(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "workbench")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	useSingleRunnerRoot(t, root)

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

func TestResolveRunnerProjectSearchesMultipleAuthorisedRoots(t *testing.T) {
	root1 := t.TempDir()
	root2 := t.TempDir()
	repo := filepath.Join(root2, "garage")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	useRunnerRoots(t, root1, root2)

	for _, requested := range []string{"runner://garage", `C:\workspace\garage`, "garage"} {
		got, err := ResolveRunnerProject(requested)
		if err != nil {
			t.Fatalf("resolve %q: %v", requested, err)
		}
		gotInfo, _ := os.Stat(got)
		repoInfo, _ := os.Stat(repo)
		if !os.SameFile(gotInfo, repoInfo) {
			t.Fatalf("resolve %q got %q want %q", requested, got, repo)
		}
	}
}

func TestResolveRunnerProjectRequiresScopedRefForDuplicateNames(t *testing.T) {
	root1 := t.TempDir()
	root2 := t.TempDir()
	repo1 := filepath.Join(root1, "shared")
	repo2 := filepath.Join(root2, "shared")
	for _, repo := range []string{repo1, repo2} {
		if err := os.MkdirAll(repo, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	useRunnerRoots(t, root1, root2)

	if _, err := ResolveRunnerProject("runner://shared"); err == nil {
		t.Fatal("expected duplicate unscoped project name to fail closed")
	}
	got, err := ResolveRunnerProject("runner://r2/shared")
	if err != nil {
		t.Fatal(err)
	}
	gotInfo, _ := os.Stat(got)
	wantInfo, _ := os.Stat(repo2)
	if !os.SameFile(gotInfo, wantInfo) {
		t.Fatalf("scoped ref resolved %q want %q", got, repo2)
	}
}

func TestResolveRunnerProjectRejectsOutsideRoots(t *testing.T) {
	root1 := t.TempDir()
	root2 := t.TempDir()
	outside := t.TempDir()
	useRunnerRoots(t, root1, root2)

	if _, err := ResolveRunnerProject(outside); err == nil {
		t.Fatal("expected outside-root path to be rejected")
	}
}

func TestResolveRunnerProjectRejectsSymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation is not guaranteed for unprivileged Windows CI")
	}
	root1 := t.TempDir()
	root2 := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(root2, "escape")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	useRunnerRoots(t, root1, root2)

	if _, err := ResolveRunnerProject("runner://r2/escape"); err == nil {
		t.Fatal("expected in-root symlink to outside directory to be rejected")
	}
	if _, err := ResolveRunnerProject(`C:\workspace\escape`); err == nil {
		t.Fatal("expected name resolution through symlink escape to fail")
	}
}

func TestRunnerRootsDefaultToSrcAndProjectsWhenPresent(t *testing.T) {
	home := t.TempDir()
	src := filepath.Join(home, "src")
	projects := filepath.Join(home, "projects")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(projects, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WORKBENCH_RUNNER_ROOT", "")
	t.Setenv("WORKBENCH_RUNNER_ROOTS", "")
	t.Setenv("HOME", home)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home)
	}

	roots, err := runnerRoots()
	if err != nil {
		t.Fatal(err)
	}
	if len(roots) != 2 {
		t.Fatalf("roots=%v want src+projects", roots)
	}
}

func TestRunnerRootSpecsKeepProjectsAsSlotTwoWhenSrcIsMissing(t *testing.T) {
	home := t.TempDir()
	projects := filepath.Join(home, "projects")
	if err := os.MkdirAll(projects, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WORKBENCH_RUNNER_ROOT", "")
	t.Setenv("WORKBENCH_RUNNER_ROOTS", "")
	t.Setenv("HOME", home)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home)
	}

	specs, err := runnerRootSpecs()
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 1 {
		t.Fatalf("specs=%+v want one existing default root", specs)
	}
	if specs[0].Number != 2 {
		t.Fatalf("spec=%+v want stable slot 2", specs[0])
	}
	gotInfo, err := os.Stat(specs[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	wantInfo, err := os.Stat(projects)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(gotInfo, wantInfo) {
		t.Fatalf("spec path %q does not identify projects root %q", specs[0].Path, projects)
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
