package desktop

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestRegisterActiveChatProjectsAddsOnlyRecentAvailableProjects(t *testing.T) {
	store, err := core.NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	eng, err := core.NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	inventory := []core.RunnerProjectInfo{
		{Name: "garage", Ref: "runner://garage"},
		{Name: "rum", Ref: "runner://rum"},
		{Name: "old-project", Ref: "runner://old-project"},
	}
	activity := []core.RunnerChatActivityInfo{
		{ID: "garage-read", ProjectRef: "runner://garage", Action: "read_file", State: "completed", UpdatedAt: now.Add(-time.Minute)},
		{ID: "rum-test", ProjectRef: "runner://rum", Action: "run_safe_command", State: "completed", UpdatedAt: now.Add(-5 * time.Minute)},
		{ID: "stale", ProjectRef: "runner://old-project", Action: "read_file", State: "completed", UpdatedAt: now.Add(-chatProjectAutoRegisterWindow - time.Second)},
		{ID: "gone", ProjectRef: "runner://removed", Action: "read_file", State: "completed", UpdatedAt: now.Add(-time.Minute)},
	}
	added, err := registerActiveChatProjects(eng, inventory, activity, now)
	if err != nil {
		t.Fatal(err)
	}
	if added != 2 {
		t.Fatalf("added=%d want 2", added)
	}
	projects := eng.Projects()
	seen := map[string]bool{}
	for _, project := range projects {
		seen[project.Path] = true
	}
	if len(projects) != 2 || !seen["runner://garage"] || !seen["runner://rum"] || seen["runner://old-project"] || seen["runner://removed"] {
		t.Fatalf("unexpected projects: %#v", projects)
	}
	added, err = registerActiveChatProjects(eng, inventory, activity, now)
	if err != nil || added != 0 {
		t.Fatalf("repeat registration added=%d err=%v", added, err)
	}
}

func TestRegisterActiveChatProjectsAcceptsScopedRunnerReference(t *testing.T) {
	store, err := core.NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	eng, err := core.NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	inventory := []core.RunnerProjectInfo{{Name: "shared", Ref: "runner://r2/shared"}}
	activity := []core.RunnerChatActivityInfo{{ID: "scoped", ProjectRef: "RUNNER://R2/shared", Action: "read_file", State: "completed", UpdatedAt: now}}
	added, err := registerActiveChatProjects(eng, inventory, activity, now)
	if err != nil {
		t.Fatal(err)
	}
	if added != 1 || len(eng.Projects()) != 1 || eng.Projects()[0].Path != "runner://r2/shared" {
		t.Fatalf("scoped project not registered: added=%d projects=%#v", added, eng.Projects())
	}
}
