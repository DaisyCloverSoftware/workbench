package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestApplyPublicationPolicyCommandRoundTrip(t *testing.T) {
	isolatePolicyConfig(t)
	repo := initPolicyRepo(t)

	result, err := applyPublicationPolicyCommand([]string{"prepare", repo})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK || result.Policy == nil || result.Policy.Mode != core.PublicationPrepare || result.Policy.RemoteURL != "" {
		t.Fatalf("unexpected prepare policy: %#v", result)
	}
	assertSamePolicyPath(t, result.Policy.Project, repo)

	getResult, err := applyPublicationPolicyCommand([]string{"get", repo})
	if err != nil {
		t.Fatal(err)
	}
	if !getResult.OK || !getResult.Configured || getResult.Policy == nil {
		t.Fatalf("prepare policy was not persisted: %#v", getResult)
	}

	remote := filepath.Join(t.TempDir(), "review.git")
	runPolicyGit(t, "init", "--bare", "-q", remote)
	result, err = applyPublicationPolicyCommand([]string{"publish", repo, remote})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK || result.Policy == nil || result.Policy.Mode != core.PublicationPublish || result.Policy.RemoteURL != remote {
		t.Fatalf("unexpected publish policy: %#v", result)
	}
	assertSamePolicyPath(t, result.Policy.Project, repo)

	deleted, err := applyPublicationPolicyCommand([]string{"delete", repo})
	if err != nil {
		t.Fatal(err)
	}
	if !deleted.OK || !deleted.Deleted {
		t.Fatalf("unexpected delete response: %#v", deleted)
	}
	getResult, err = applyPublicationPolicyCommand([]string{"get", repo})
	if err != nil {
		t.Fatal(err)
	}
	if getResult.Configured || getResult.Policy != nil {
		t.Fatalf("policy was not deleted: %#v", getResult)
	}
}

func TestApplyPublicationPolicyCommandMapsDesktopProjectPath(t *testing.T) {
	isolatePolicyConfig(t)
	repo := initPolicyRepo(t)

	result, err := applyPublicationPolicyCommand([]string{"prepare", `C:\workspace\workbench`})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK || result.Policy == nil || result.Policy.Mode != core.PublicationPrepare {
		t.Fatalf("desktop project did not map to runner repository: %#v", result)
	}
	assertSamePolicyPath(t, result.Policy.Project, repo)
}

func TestApplyPublicationPolicyCommandRejectsUnsafePublishTarget(t *testing.T) {
	isolatePolicyConfig(t)
	repo := initPolicyRepo(t)
	if _, err := applyPublicationPolicyCommand([]string{"publish", repo, "https://user:secret@example.invalid/repo.git"}); err == nil {
		t.Fatal("expected embedded credentials to be rejected")
	}
}

func isolatePolicyConfig(t *testing.T) {
	t.Helper()
	config := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", config)
	t.Setenv("APPDATA", config)
}

func initPolicyRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	repo := filepath.Join(root, "workbench")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WORKBENCH_RUNNER_ROOT", root)
	runPolicyGit(t, "-C", repo, "init", "-q")
	runPolicyGit(t, "-C", repo, "config", "user.name", "Workbench Test")
	runPolicyGit(t, "-C", repo, "config", "user.email", "workbench-test@example.invalid")
	if err := os.WriteFile(filepath.Join(repo, "tracked.txt"), []byte("baseline\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runPolicyGit(t, "-C", repo, "add", "tracked.txt")
	runPolicyGit(t, "-C", repo, "commit", "-q", "-m", "baseline")
	return repo
}

func assertSamePolicyPath(t *testing.T, got, want string) {
	t.Helper()
	gotInfo, err := os.Stat(got)
	if err != nil {
		t.Fatalf("stat policy project %q: %v", got, err)
	}
	wantInfo, err := os.Stat(want)
	if err != nil {
		t.Fatalf("stat expected project %q: %v", want, err)
	}
	if !os.SameFile(gotInfo, wantInfo) {
		t.Fatalf("policy project %q does not identify expected repository %q", got, want)
	}
}

func runPolicyGit(t *testing.T, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
