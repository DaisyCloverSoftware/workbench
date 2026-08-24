package core

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
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

func TestRunnerGitRemoteHeadsEquivalentRequiresCompleteMatchingHeadSet(t *testing.T) {
	base := t.TempDir()
	seed := filepath.Join(base, "seed")
	if out, err := exec.Command("git", "init", "--quiet", seed).CombinedOutput(); err != nil {
		t.Fatalf("init seed: %v: %s", err, out)
	}
	for _, args := range [][]string{{"config", "user.name", "Workbench Test"}, {"config", "user.email", "workbench-test@example.invalid"}} {
		if out, err := exec.Command("git", append([]string{"-C", seed}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("configure seed: %v: %s", err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(seed, "README.md"), []byte("same repository\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "README.md"}, {"commit", "--quiet", "-m", "initial"}, {"branch", "-M", "main"}} {
		if out, err := exec.Command("git", append([]string{"-C", seed}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("seed commit: %v: %s", err, out)
		}
	}

	bareA := filepath.Join(base, "origin-a.git")
	bareB := filepath.Join(base, "origin-b.git")
	for _, target := range []string{bareA, bareB} {
		if out, err := exec.Command("git", "clone", "--quiet", "--bare", seed, target).CombinedOutput(); err != nil {
			t.Fatalf("clone bare %s: %v: %s", target, err, out)
		}
	}
	project := filepath.Join(base, "project")
	if out, err := exec.Command("git", "clone", "--quiet", bareA, project).CombinedOutput(); err != nil {
		t.Fatalf("clone project: %v: %s", err, out)
	}

	if !runnerGitRemoteHeadsEquivalent(context.Background(), project, bareB) {
		t.Fatal("identical complete branch-ref sets must prove equivalence")
	}

	if out, err := exec.Command("git", "-C", seed, "checkout", "--quiet", "-b", "extra").CombinedOutput(); err != nil {
		t.Fatalf("create extra branch: %v: %s", err, out)
	}
	if err := os.WriteFile(filepath.Join(seed, "EXTRA.md"), []byte("extra branch\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "EXTRA.md"}, {"commit", "--quiet", "-m", "extra"}, {"push", "--quiet", bareB, "extra:refs/heads/extra"}} {
		if out, err := exec.Command("git", append([]string{"-C", seed}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("publish divergent branch: %v: %s", err, out)
		}
	}
	if runnerGitRemoteHeadsEquivalent(context.Background(), project, bareB) {
		t.Fatal("different complete branch-ref sets must not prove equivalence")
	}
}

func TestParseGitRemoteHeadsRejectsMalformedOrEmptyInput(t *testing.T) {
	valid := strings.Repeat("a", 40) + "\trefs/heads/main\n" + strings.Repeat("b", 40) + "\trefs/heads/release"
	heads, ok := parseGitRemoteHeads(valid)
	if !ok || len(heads) != 2 || heads["refs/heads/main"] != strings.Repeat("a", 40) {
		t.Fatalf("valid heads were not parsed: ok=%v heads=%v", ok, heads)
	}
	for _, raw := range []string{
		"",
		"not-a-sha\trefs/heads/main",
		strings.Repeat("a", 40) + "\trefs/tags/v1",
		strings.Repeat("a", 40) + "\trefs/heads/main\n" + strings.Repeat("b", 40) + "\trefs/heads/main",
	} {
		if _, ok := parseGitRemoteHeads(raw); ok {
			t.Fatalf("malformed head set accepted: %q", raw)
		}
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
