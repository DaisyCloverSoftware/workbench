//go:build windows

package core

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInspectOverrideRinWardrobeRootIsReadOnly(t *testing.T) {
	gitExecutable, err := exec.LookPath("git.exe")
	if err != nil {
		t.Skip("Git for Windows is unavailable")
	}
	root := t.TempDir()
	repo := filepath.Join(root, "override-hardline")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	runWardrobeTestGit(t, gitExecutable, repo, "init")
	runWardrobeTestGit(t, gitExecutable, repo, "config", "user.name", "Workbench Test")
	runWardrobeTestGit(t, gitExecutable, repo, "config", "user.email", "workbench-test@example.invalid")
	runWardrobeTestGit(t, gitExecutable, repo, "checkout", "-b", overrideRinWardrobeBranch)
	runWardrobeTestGit(t, gitExecutable, repo, "remote", "add", "origin", "https://github.com/DaisyCloverSoftware/override.git")

	tracked := filepath.Join(repo, "Content", "Wardrobe", "Cargo_Trousers.uasset")
	untracked := filepath.Join(repo, "SourceAssets", "RinVale", "Clash_Shirt.blend")
	ignored := filepath.Join(repo, "Content", "Environment", "FactoryWall.uasset")
	for path, data := range map[string][]byte{
		tracked:   []byte("tracked cargo trousers fixture\n"),
		untracked: []byte("untracked clash shirt fixture\n"),
		ignored:   []byte("unrelated environment fixture\n"),
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	runWardrobeTestGit(t, gitExecutable, repo, "add", filepath.ToSlash(filepath.Join("Content", "Wardrobe", "Cargo_Trousers.uasset")))
	runWardrobeTestGit(t, gitExecutable, repo, "commit", "-m", "fixture")

	before := wardrobeTestStatus(t, gitExecutable, repo)
	result, err := inspectOverrideRinWardrobeRoot(context.Background(), gitExecutable, root, "hostjob_wardrobe_test")
	if err != nil {
		t.Fatal(err)
	}
	after := wardrobeTestStatus(t, gitExecutable, repo)
	if before != after {
		t.Fatalf("scanner changed worktree status: before=%q after=%q", before, after)
	}
	if !result.ReadOnly || result.MutationPerformed {
		t.Fatalf("bad mutation boundary: %+v", result)
	}
	if result.Branch != overrideRinWardrobeBranch || len(result.HeadSHA) != 40 {
		t.Fatalf("bad worktree identity: %+v", result)
	}
	if result.VisualAcceptance != "not_assessed" || result.AnimationAcceptance != "not_assessed" {
		t.Fatalf("scanner crossed acceptance boundary: %+v", result)
	}

	byPath := map[string]overrideRinWardrobeCandidate{}
	for _, candidate := range result.Candidates {
		byPath[candidate.Path] = candidate
	}
	cargo, ok := byPath["Content/Wardrobe/Cargo_Trousers.uasset"]
	if !ok || !cargo.Tracked || cargo.HashStatus != "complete" || len(cargo.SHA256) != 64 {
		t.Fatalf("tracked cargo fixture missing or invalid: %+v", cargo)
	}
	shirt, ok := byPath["SourceAssets/RinVale/Clash_Shirt.blend"]
	if !ok || shirt.Tracked || shirt.HashStatus != "complete" || len(shirt.SHA256) != 64 {
		t.Fatalf("untracked shirt fixture missing or invalid: %+v", shirt)
	}
	for path := range byPath {
		if strings.Contains(path, "FactoryWall") {
			t.Fatalf("unrelated asset leaked into wardrobe inventory: %q", path)
		}
	}
}

func runWardrobeTestGit(t *testing.T, gitExecutable, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command(gitExecutable, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0", "GIT_TERMINAL_PROMPT=0", "GCM_INTERACTIVE=Never")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v: %s", args, err, output)
	}
}

func wardrobeTestStatus(t *testing.T, gitExecutable, dir string) string {
	t.Helper()
	cmd := exec.Command(gitExecutable, "status", "--porcelain=v1", "--untracked-files=all")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0", "GIT_TERMINAL_PROMPT=0", "GCM_INTERACTIVE=Never")
	output, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	return string(output)
}
