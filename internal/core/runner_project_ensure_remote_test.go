package core

import (
	"context"
	"errors"
	"os/exec"
	"testing"
)

func TestEnsureRunnerGitHubProjectReusesHistoricalLocalDirectoryByRemote(t *testing.T) {
	root := t.TempDir()
	useRunnerRoots(t, root)
	repo := initTestRepo(t, root, "rum-alpha")
	if out, err := exec.Command("git", "-C", repo, "remote", "add", "origin", "git@github.com:DaisyCloverSoftware/rum.git").CombinedOutput(); err != nil {
		t.Fatalf("add origin: %v: %s", err, out)
	}
	called := false
	project, cloned, err := ensureRunnerGitHubProject(context.Background(), "DaisyCloverSoftware/rum", func(context.Context, string, string) error {
		called = true
		return errors.New("must not clone")
	})
	if err != nil {
		t.Fatal(err)
	}
	if called || cloned || project.Name != "rum-alpha" || project.Ref != "runner://rum-alpha" {
		t.Fatalf("historical checkout was not reused: called=%v cloned=%v project=%+v", called, cloned, project)
	}
}

func TestGitHubSlugFromRemoteAcceptsFixedGitHubFormsOnly(t *testing.T) {
	for remote, want := range map[string]string{
		"git@github.com:ExampleOrg/repo.git":       "exampleorg/repo",
		"ssh://git@github.com/ExampleOrg/repo.git": "exampleorg/repo",
		"https://github.com/ExampleOrg/repo.git":    "exampleorg/repo",
	} {
		got, ok := githubSlugFromRemote(remote)
		if !ok || got != want {
			t.Fatalf("githubSlugFromRemote(%q)=(%q,%v) want %q,true", remote, got, ok, want)
		}
	}
	if _, ok := githubSlugFromRemote("https://example.invalid/ExampleOrg/repo.git"); ok {
		t.Fatal("non-GitHub remote must not be treated as a trusted GitHub slug")
	}
}
