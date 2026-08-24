package core

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func TestEnsureRunnerGitHubProjectNormalisesLegacyCredentialBearingOrigin(t *testing.T) {
	root := t.TempDir()
	useRunnerRoots(t, root)
	repo := initTestRepo(t, root, "infrastructure")
	legacy := "https://runner-user@github.com/DaisyCloverSoftware/infrastructure.git"
	if out, err := exec.Command("git", "-C", repo, "remote", "add", "origin", legacy).CombinedOutput(); err != nil {
		t.Fatalf("add legacy origin: %v: %s", err, out)
	}
	if isApprovedOperationsOrigin(legacy) {
		t.Fatal("legacy userinfo origin must not be approved for exact-commit operations")
	}

	called := false
	project, cloned, err := ensureRunnerGitHubProject(context.Background(), "DaisyCloverSoftware/infrastructure", func(context.Context, string, string) error {
		called = true
		return errors.New("must not clone")
	})
	if err != nil {
		t.Fatal(err)
	}
	if called || cloned || project.Name != "infrastructure" || project.Ref != "runner://infrastructure" {
		t.Fatalf("existing checkout was not reused: called=%v cloned=%v project=%+v", called, cloned, project)
	}
	out, err := exec.Command("git", "-C", repo, "remote", "get-url", "origin").CombinedOutput()
	if err != nil {
		t.Fatalf("read normalised origin: %v: %s", err, out)
	}
	got := strings.TrimSpace(string(out))
	want := "git@github.com:DaisyCloverSoftware/infrastructure.git"
	if got != want {
		t.Fatalf("normalised origin=%q want %q", got, want)
	}
	if !isApprovedOperationsOrigin(got) {
		t.Fatal("normalised origin must satisfy exact-commit operations policy")
	}
}

func TestEnsureRunnerGitHubProjectNormalisesInsteadOfCredentialRewrite(t *testing.T) {
	root := t.TempDir()
	useRunnerRoots(t, root)
	repo := initTestRepo(t, root, "infrastructure")
	configured := "https://github.com/DaisyCloverSoftware/infrastructure.git"
	if out, err := exec.Command("git", "-C", repo, "remote", "add", "origin", configured).CombinedOutput(); err != nil {
		t.Fatalf("add configured origin: %v: %s", err, out)
	}
	if out, err := exec.Command("git", "-C", repo, "config", "url.https://runner-user@github.com/.insteadOf", "https://github.com/").CombinedOutput(); err != nil {
		t.Fatalf("add credential rewrite: %v: %s", err, out)
	}

	effectiveOut, err := exec.Command("git", "-C", repo, "remote", "get-url", "origin").CombinedOutput()
	if err != nil {
		t.Fatalf("read rewritten origin: %v: %s", err, effectiveOut)
	}
	effective := strings.TrimSpace(string(effectiveOut))
	if effective == configured || isApprovedOperationsOrigin(effective) {
		t.Fatalf("test must exercise a rewritten non-approved effective origin")
	}
	rawOut, err := exec.Command("git", "-C", repo, "config", "--get", "remote.origin.url").CombinedOutput()
	if err != nil || strings.TrimSpace(string(rawOut)) != configured {
		t.Fatalf("configured origin identity was not preserved: err=%v", err)
	}

	called := false
	project, cloned, err := ensureRunnerGitHubProject(context.Background(), "DaisyCloverSoftware/infrastructure", func(context.Context, string, string) error {
		called = true
		return errors.New("must not clone")
	})
	if err != nil {
		t.Fatal(err)
	}
	if called || cloned || project.Ref != "runner://infrastructure" {
		t.Fatalf("rewritten checkout was not safely reused: called=%v cloned=%v project=%+v", called, cloned, project)
	}
	out, err := exec.Command("git", "-C", repo, "remote", "get-url", "origin").CombinedOutput()
	if err != nil {
		t.Fatalf("read normalised effective origin: %v: %s", err, out)
	}
	got := strings.TrimSpace(string(out))
	want := "git@github.com:DaisyCloverSoftware/infrastructure.git"
	if got != want || !isApprovedOperationsOrigin(got) {
		t.Fatalf("normalised effective origin=%q want approved %q", got, want)
	}
}

func TestGitHubSlugFromRemoteForEnsureDoesNotExposeOrTrustOtherHosts(t *testing.T) {
	got, ok := githubSlugFromRemoteForEnsure("https://runner-user@github.com/ExampleOrg/repo.git")
	if !ok || got != "exampleorg/repo" {
		t.Fatalf("credential-bearing GitHub identity=(%q,%v), want exampleorg/repo,true", got, ok)
	}
	for _, remote := range []string{
		"https://runner-user@example.invalid/ExampleOrg/repo.git",
		"https://runner-user@github.com/ExampleOrg/repo.git?ref=main",
		"https://runner-user@github.com/ExampleOrg/repo.git#fragment",
	} {
		if _, ok := githubSlugFromRemoteForEnsure(remote); ok {
			t.Fatalf("untrusted or decorated remote must be rejected: %q", remote)
		}
	}
}
