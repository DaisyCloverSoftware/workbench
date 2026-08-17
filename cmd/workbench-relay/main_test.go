package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidRelayID(t *testing.T) {
	for _, id := range []string{"wb_20260812_abc123", "task-12345678", "Relay_OK_123"} {
		if !validRelayID(id) {
			t.Fatalf("expected valid relay id %q", id)
		}
	}
	for _, id := range []string{"short", "../oops", "has space 123", "a/b/cdefgh"} {
		if validRelayID(id) {
			t.Fatalf("expected invalid relay id %q", id)
		}
	}
}

func TestResolveProjectStaysUnderRunnerRoot(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "workbench")
	if err := os.Mkdir(project, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WORKBENCH_RUNNER_ROOTS", "")
	t.Setenv("WORKBENCH_RUNNER_ROOT", root)
	got, err := resolveProject("workbench")
	if err != nil {
		t.Fatal(err)
	}
	if got != project {
		t.Fatalf("got %q want %q", got, project)
	}
	for _, bad := range []string{"../workbench", "/tmp/workbench", "sub/workbench", `sub\\workbench`} {
		if _, err := resolveProject(bad); err == nil {
			t.Fatalf("expected project %q to be rejected", bad)
		}
	}
}

func TestResolveProjectUsesScopedReferenceAcrossMultipleRoots(t *testing.T) {
	root1 := t.TempDir()
	root2 := t.TempDir()
	left := filepath.Join(root1, "shared")
	right := filepath.Join(root2, "shared")
	if err := os.Mkdir(left, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(right, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WORKBENCH_RUNNER_ROOT", "")
	t.Setenv("WORKBENCH_RUNNER_ROOTS", strings.Join([]string{root1, root2}, string(os.PathListSeparator)))

	if _, err := resolveProject("shared"); err == nil {
		t.Fatal("duplicate project name must require scoped reference")
	}
	got, err := resolveProject("runner://r2/shared")
	if err != nil {
		t.Fatal(err)
	}
	gotInfo, _ := os.Stat(got)
	rightInfo, _ := os.Stat(right)
	if !os.SameFile(gotInfo, rightInfo) {
		t.Fatalf("scoped relay project got %q want %q", got, right)
	}
}
