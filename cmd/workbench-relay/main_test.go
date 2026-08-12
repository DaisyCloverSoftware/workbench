package main

import (
	"os"
	"path/filepath"
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
