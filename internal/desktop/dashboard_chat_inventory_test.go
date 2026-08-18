package desktop

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestFilterRunnerChatActivityDropsUnavailableRunnerProjects(t *testing.T) {
	now := time.Now().UTC()
	inventory := []core.RunnerProjectInfo{{Name: "workbench", Ref: "runner://workbench"}}
	activity := []core.RunnerChatActivityInfo{
		{ID: "canonical", ProjectRef: "runner://workbench", Action: "read_file", UpdatedAt: now},
		{ID: "worktree", ProjectRef: "runner://workbench-fix-review", Action: "read_file", UpdatedAt: now},
	}
	got := filterRunnerChatActivityToInventory(inventory, activity)
	if len(got) != 1 || got[0].ID != "canonical" {
		t.Fatalf("filtered activity=%+v", got)
	}
}

func TestPruneUnavailableRunnerProjectsRemovesOnlyEmptyUnpinnedEntries(t *testing.T) {
	store, err := core.NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	eng, err := core.NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := eng.RegisterRunnerProjects([]core.RunnerProjectInfo{
		{Name: "workbench", Ref: "runner://workbench"},
		{Name: "workbench-fix-review", Ref: "runner://workbench-fix-review"},
		{Name: "kept-pinned", Ref: "runner://kept-pinned"},
	}); err != nil {
		t.Fatal(err)
	}
	for _, project := range eng.Projects() {
		if project.Path == "runner://kept-pinned" {
			if err := eng.SetProjectPinned(project.ID, true); err != nil {
				t.Fatal(err)
			}
		}
	}

	removed := pruneUnavailableRunnerProjects(eng, []core.RunnerProjectInfo{{Name: "workbench", Ref: "runner://workbench"}})
	if removed != 1 {
		t.Fatalf("removed=%d want 1", removed)
	}
	paths := map[string]bool{}
	for _, project := range eng.Projects() {
		paths[project.Path] = true
	}
	if paths["runner://workbench-fix-review"] {
		t.Fatalf("temporary runner project survived pruning: %+v", paths)
	}
	if !paths["runner://workbench"] || !paths["runner://kept-pinned"] {
		t.Fatalf("canonical or pinned project was pruned: %+v", paths)
	}
}

func TestPruneUnavailableRunnerProjectsDoesNothingFromEmptyInventory(t *testing.T) {
	store, err := core.NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	eng, err := core.NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := eng.RegisterRunnerProjects([]core.RunnerProjectInfo{{Name: "workbench", Ref: "runner://workbench"}}); err != nil {
		t.Fatal(err)
	}
	if removed := pruneUnavailableRunnerProjects(eng, nil); removed != 0 {
		t.Fatalf("removed=%d from empty inventory", removed)
	}
	if len(eng.Projects()) != 1 {
		t.Fatalf("empty inventory pruned registered project: %+v", eng.Projects())
	}
}
