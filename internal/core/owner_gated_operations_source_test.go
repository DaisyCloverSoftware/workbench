package core

import (
	"context"
	"os/exec"
	"strings"
	"testing"
)

func TestOwnerGatedRUMOriginRecognisesSupportedGitHubForms(t *testing.T) {
	for _, origin := range []string{
		"https://github.com/DaisyCloverSoftware/rum.git",
		"git@github.com:DaisyCloverSoftware/rum.git",
		"ssh://git@github.com/DaisyCloverSoftware/rum.git",
	} {
		if !ownerGatedRUMOrigin(origin) {
			t.Fatalf("expected RUM origin %q to be protected", origin)
		}
	}
	if ownerGatedRUMOrigin("https://github.com/DaisyCloverSoftware/workbench.git") {
		t.Fatal("expected unrelated repository origin to remain unprotected")
	}
}

func TestValidateOwnerGatedOperationsSourceRequiresExactRUMMain(t *testing.T) {
	root := initOperationsSourceTestRepo(t, "https://github.com/DaisyCloverSoftware/rum.git")
	mainCommit := strings.Repeat("a", 40)
	branchCommit := strings.Repeat("b", 40)

	previous := ownerGatedOperationsMainHead
	ownerGatedOperationsMainHead = func(context.Context, string) (string, error) { return mainCommit, nil }
	t.Cleanup(func() { ownerGatedOperationsMainHead = previous })

	if err := validateOwnerGatedOperationsSource(context.Background(), root, mainCommit); err != nil {
		t.Fatalf("expected exact RUM main to remain eligible, got %v", err)
	}
	if err := validateOwnerGatedOperationsSource(context.Background(), root, branchCommit); err == nil || !strings.Contains(err.Error(), "exact current main") {
		t.Fatalf("expected RUM branch-head operations source to fail closed, got %v", err)
	}
}

func TestValidateOwnerGatedOperationsSourceLeavesOtherRepositoriesUnchanged(t *testing.T) {
	root := initOperationsSourceTestRepo(t, "https://github.com/DaisyCloverSoftware/workbench.git")
	previous := ownerGatedOperationsMainHead
	ownerGatedOperationsMainHead = func(context.Context, string) (string, error) {
		t.Fatal("main resolver must not run for unrelated repositories")
		return "", nil
	}
	t.Cleanup(func() { ownerGatedOperationsMainHead = previous })

	if err := validateOwnerGatedOperationsSource(context.Background(), root, strings.Repeat("b", 40)); err != nil {
		t.Fatalf("expected unrelated repository operations source to remain eligible, got %v", err)
	}
}

func initOperationsSourceTestRepo(t *testing.T, origin string) string {
	t.Helper()
	root := t.TempDir()
	for _, args := range [][]string{
		{"init", "--quiet", root},
		{"-C", root, "remote", "add", "origin", origin},
	} {
		cmd := exec.Command("git", args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v: %s", args, err, strings.TrimSpace(string(out)))
		}
	}
	return root
}
