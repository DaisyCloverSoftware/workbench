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
	prepared, ok := result["policy"].(core.PublicationPolicy)
	if !ok || prepared.Mode != core.PublicationPrepare || prepared.RemoteURL != "" {
		t.Fatalf("unexpected prepare policy: %#v", result)
	}
	assertSamePolicyPath(t, prepared.Project, repo)

	getResult, err := applyPublicationPolicyCommand([]string{"get", repo})
	if err != nil {
		t.Fatal(err)
	}
	if configured, _ := getResult["configured"].(bool); !configured {
		t.Fatalf("prepare policy was not persisted: %#v", getResult)
	}

	remote := filepath.Join(t.TempDir(), "review.git")
	runPolicyGit(t, "init", "--bare", "-q", remote)
	result, err = applyPublicationPolicyCommand([]string{"publish", repo, remote})
	if err != nil {
		t.Fatal(err)
	}
	published, ok := result["policy"].(core.PublicationPolicy)
	if !ok || published.Mode != core.PublicationPublish || published.RemoteURL != remote {
		t.Fatalf("unexpected publish policy: %#v", result)
	}
	assertSamePolicyPath(t, published.Project, repo)

	if _, err := applyPublicationPolicyCommand([]string{"delete", repo}); err != nil {
		t.Fatal(err)
	}
	getResult, err = applyPublicationPolicyCommand([]string{"get", repo})
	if err != nil {
		t.Fatal(err)
	}
	if configured, _ := getResult["configured"].(bool); configured {
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
	prepared, ok := result["policy"].(core.PublicationPolicy)
	if !ok || prepared.Mode != core.PublicationPrepare {
		t.Fatalf("desktop project did not map to runner repository: %#v", result)
	}
	assertSamePolicyPath(t, prepared.Project, repo)
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
