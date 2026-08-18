package core

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestEnsureRunnerGitHubProjectReturnsExistingRepoWithoutClone(t *testing.T) {
	root := t.TempDir()
	useRunnerRoots(t, root)
	initTestRepo(t, root, "garage")
	called := false
	project, cloned, err := ensureRunnerGitHubProject(context.Background(), "DaisyCloverSoftware/garage", func(context.Context, string, string) error {
		called = true
		return errors.New("must not clone")
	})
	if err != nil {
		t.Fatal(err)
	}
	if called || cloned || project.Name != "garage" || project.Ref != "runner://garage" {
		t.Fatalf("unexpected existing-project result: called=%v cloned=%v project=%+v", called, cloned, project)
	}
}

func TestEnsureRunnerGitHubProjectClonesMissingRepoAtomically(t *testing.T) {
	root := t.TempDir()
	useRunnerRoots(t, root)
	clone := func(_ context.Context, repository, target string) error {
		if repository != "DaisyCloverSoftware/override" {
			t.Fatalf("repository=%q", repository)
		}
		if out, err := exec.Command("git", "init", target).CombinedOutput(); err != nil {
			t.Fatalf("git init: %v: %s", err, out)
		}
		return os.WriteFile(filepath.Join(target, "README.md"), []byte("override\n"), 0o600)
	}
	project, cloned, err := ensureRunnerGitHubProject(context.Background(), "DaisyCloverSoftware/override", clone)
	if err != nil {
		t.Fatal(err)
	}
	if !cloned || project.Name != "override" || project.Ref != "runner://override" {
		t.Fatalf("unexpected cloned-project result: cloned=%v project=%+v", cloned, project)
	}
	if _, err := os.Stat(filepath.Join(root, "override", "README.md")); err != nil {
		t.Fatalf("installed clone missing: %v", err)
	}
	matches, err := filepath.Glob(filepath.Join(root, ".workbench-clone-override-*"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("temporary clone directory leaked: %v %v", matches, err)
	}
}

func TestEnsureRunnerGitHubProjectRejectsArbitraryRemoteAndExistingCollision(t *testing.T) {
	root := t.TempDir()
	useRunnerRoots(t, root)
	for _, bad := range []string{"https://example.com/repo", "owner/repo.git", "owner/repo/extra", "owner/repo;touch-x"} {
		if _, _, err := ensureRunnerGitHubProject(context.Background(), bad, nil); err == nil {
			t.Fatalf("unsafe repository slug %q accepted", bad)
		}
	}
	if err := os.Mkdir(filepath.Join(root, "rum"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ensureRunnerGitHubProject(context.Background(), "DaisyCloverSoftware/rum", func(context.Context, string, string) error {
		return nil
	}); err == nil {
		t.Fatal("existing non-repository destination must fail closed")
	}
}
