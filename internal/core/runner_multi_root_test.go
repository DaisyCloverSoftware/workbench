package core

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func initTestRepo(t *testing.T, root, name string) string {
	t.Helper()
	repo := filepath.Join(root, name)
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "init", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init %s: %v: %s", repo, err, out)
	}
	return repo
}

func TestRunnerToolDiscoversMultipleRootsAndScopesDuplicateNames(t *testing.T) {
	root1 := t.TempDir()
	root2 := t.TempDir()
	initTestRepo(t, root1, "src-only")
	initTestRepo(t, root2, "projects-only")
	initTestRepo(t, root1, "shared")
	initTestRepo(t, root2, "shared")
	useRunnerRoots(t, root1, root2)

	response, err := ApplyRunnerToolRequest(context.Background(), RunnerToolRequest{Action: "list_projects"})
	if err != nil {
		t.Fatal(err)
	}
	if !response.OK || len(response.Projects) != 4 {
		t.Fatalf("unexpected multi-root response: %+v", response)
	}
	refs := map[string]bool{}
	for _, project := range response.Projects {
		refs[project.Ref] = true
	}
	for _, want := range []string{
		"runner://src-only",
		"runner://projects-only",
		"runner://r1/shared",
		"runner://r2/shared",
	} {
		if !refs[want] {
			t.Fatalf("multi-root discovery missing %q: %+v", want, response.Projects)
		}
	}
}

func TestScopedRunnerProjectsRemainDistinctInRegistry(t *testing.T) {
	left := "runner://r1/shared"
	right := "runner://r2/shared"
	if got := normalizeProjectPath(left); got != left {
		t.Fatalf("normalize left=%q want %q", got, left)
	}
	if got := normalizeProjectPath(right); got != right {
		t.Fatalf("normalize right=%q want %q", got, right)
	}
	if projectID(left) == projectID(right) {
		t.Fatal("scoped runner project references must have distinct registry identities")
	}
	if sameProjectPath(left, right) {
		t.Fatal("different authorised runner roots must not collapse to one project")
	}
}

func TestRegisterRunnerProjectsAcceptsScopedRefs(t *testing.T) {
	store, err := NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	eng, err := NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	added, err := eng.RegisterRunnerProjects([]RunnerProjectInfo{
		{Name: "shared", Ref: "runner://r1/shared"},
		{Name: "shared", Ref: "runner://r2/shared"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if added != 2 {
		t.Fatalf("added=%d want 2", added)
	}
	if len(eng.State().Projects) != 2 {
		t.Fatalf("projects=%+v", eng.State().Projects)
	}
}
