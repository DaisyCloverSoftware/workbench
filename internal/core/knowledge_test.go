package core

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestIdentifyProjectUsesPortableGitRemote(t *testing.T) {
	project := newKnowledgeTestProject(t, "https://example.invalid/acme/widget.git")
	id, err := IdentifyProject(project)
	if err != nil {
		t.Fatal(err)
	}
	if !id.Portable {
		t.Fatal("expected git-backed project identity to be portable")
	}
	if id.Key != "git:example.invalid/acme/widget" {
		t.Fatalf("project key = %q", id.Key)
	}
}

func TestRecallCombinesProjectAndGlobalMemoryWithoutCrossProjectLeak(t *testing.T) {
	store, err := NewKnowledgeStoreAt(filepath.Join(t.TempDir(), "knowledge.json"))
	if err != nil {
		t.Fatal(err)
	}
	one := newKnowledgeTestProject(t, "https://example.invalid/acme/one.git")
	two := newKnowledgeTestProject(t, "https://example.invalid/acme/two.git")
	if _, err := store.Remember(one, MemoryProject, MemoryDecision, "Use SQLite locally", "Project one stores local metadata in SQLite.", "", []string{"storage", "sqlite"}, "test"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Remember("", MemoryGlobal, MemoryPattern, "Prefer atomic writes", "Write state through a temporary file and rename it.", "", []string{"storage", "atomic"}, "test"); err != nil {
		t.Fatal(err)
	}

	gotOne, err := store.Recall(one, "storage sqlite atomic", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(gotOne) != 2 {
		t.Fatalf("project one recall count = %d, want 2", len(gotOne))
	}
	gotTwo, err := store.Recall(two, "storage sqlite atomic", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(gotTwo) != 1 || gotTwo[0].Scope != MemoryGlobal {
		t.Fatalf("project two should see only global memory: %#v", gotTwo)
	}
}

func TestCheckpointAndRoutineRenderIntoCompactContext(t *testing.T) {
	store, err := NewKnowledgeStoreAt(filepath.Join(t.TempDir(), "knowledge.json"))
	if err != nil {
		t.Fatal(err)
	}
	project := newKnowledgeTestProject(t, "https://example.invalid/acme/context.git")
	if _, err := store.SaveCheckpoint(project,
		"The relay is bidirectional and the next slice is durable memory.",
		[]string{"Keep public relay status-only."},
		[]string{"Private relay onboarding still needs dogfood."},
		[]string{"Implement compact context packs."},
	); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Remember(project, MemoryProject, MemoryConstraint, "No browser automation", "Consumer AI websites are not an integration transport.", "", []string{"integration"}, "test"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveRoutine("", MemoryGlobal, "Atomic JSON state", "Persist a small JSON state file without partial writes.", []string{"json state", "persistence"}, []string{"Marshal the new state.", "Write a sibling temporary file.", "Rename the temporary file over the state file."}, "", "", []string{"storage"}); err != nil {
		t.Fatal(err)
	}
	pack, err := store.BuildContextPack(project, "persistence integration", 10, 12000)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"CURRENT CHECKPOINT", "No browser automation", "Atomic JSON state", "Implement compact context packs"} {
		if !strings.Contains(pack.ContextText, want) {
			t.Fatalf("context pack missing %q:\n%s", want, pack.ContextText)
		}
	}
	if len(pack.ContextText) > 12000 {
		t.Fatalf("context pack exceeded budget: %d", len(pack.ContextText))
	}
}

func TestRoutineUpsertAvoidsRebuildingSameRoutine(t *testing.T) {
	store, err := NewKnowledgeStoreAt(filepath.Join(t.TempDir(), "knowledge.json"))
	if err != nil {
		t.Fatal(err)
	}
	first, err := store.SaveRoutine("", MemoryGlobal, "Go test loop", "Run the standard verification loop.", []string{"go test"}, []string{"Run go test ./..."}, "", "go", []string{"go"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.SaveRoutine("", MemoryGlobal, "Go test loop", "Run tests and then inspect the diff.", []string{"go test"}, []string{"Run go test ./...", "Inspect git diff."}, "", "go", []string{"go"})
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Fatalf("routine was duplicated: %s != %s", first.ID, second.ID)
	}
	got, err := store.FindRoutines("", "go test", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !strings.Contains(got[0].Description, "inspect the diff") {
		t.Fatalf("routine update missing: %#v", got)
	}
}

func TestKnowledgeRejectsSecretLikeMaterial(t *testing.T) {
	store, err := NewKnowledgeStoreAt(filepath.Join(t.TempDir(), "knowledge.json"))
	if err != nil {
		t.Fatal(err)
	}
	secretLike := "sk-" + "proj-" + strings.Repeat("x", 48)
	if _, err := store.Remember("", MemoryGlobal, MemoryPattern, "Bad memory", "Do not persist this: "+secretLike, "", nil, "test"); err == nil {
		t.Fatal("expected secret-like memory to be rejected")
	}
	if _, err := store.SaveRoutine("", MemoryGlobal, "Bad routine", "Contains unsafe material", nil, nil, secretLike, "text", nil); err == nil {
		t.Fatal("expected secret-like routine to be rejected")
	}
}

func TestRecordTaskOutcomeCreatesSearchableProjectMemory(t *testing.T) {
	store, err := NewKnowledgeStoreAt(filepath.Join(t.TempDir(), "knowledge.json"))
	if err != nil {
		t.Fatal(err)
	}
	project := newKnowledgeTestProject(t, "https://example.invalid/acme/outcome.git")
	task := Task{ID: "task-test", Title: "Fix relay race", ProjectPath: project, Status: TaskCompleted, Output: "Made outbox publication idempotent across platforms.", UpdatedAt: time.Now()}
	if err := store.RecordTaskOutcome(task); err != nil {
		t.Fatal(err)
	}
	got, err := store.Recall(project, "idempotent relay", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Kind != MemoryOutcome || got[0].SourceTaskID != task.ID {
		t.Fatalf("unexpected task outcome memory: %#v", got)
	}
}

func newKnowledgeTestProject(t *testing.T, remote string) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for portable project identity tests")
	}
	project := t.TempDir()
	runKnowledgeGit(t, "-C", project, "init", "--quiet")
	runKnowledgeGit(t, "-C", project, "remote", "add", "origin", remote)
	return project
}

func runKnowledgeGit(t *testing.T, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}
