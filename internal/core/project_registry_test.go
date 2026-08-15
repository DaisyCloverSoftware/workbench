package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreLoadMigratesLegacyProjectAndTaskHistory(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	active := filepath.Join(t.TempDir(), "active-project")
	historic := filepath.Join(t.TempDir(), "historic-project")
	legacy := State{
		Version:     2,
		ProjectPath: active,
		Notes:       "legacy active notes",
		Tasks: []Task{{
			ID:          "historic-task",
			ProjectPath: historic,
			CreatedAt:   time.Now().Add(-2 * time.Hour),
			UpdatedAt:   time.Now().Add(-time.Hour),
		}},
	}
	b, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(statePath, b, 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := NewStoreAt(statePath)
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != projectStateVersion {
		t.Fatalf("state version=%d want %d", got.Version, projectStateVersion)
	}
	if len(got.Projects) != 2 {
		t.Fatalf("migrated projects=%d want 2: %#v", len(got.Projects), got.Projects)
	}
	if got.ProjectPath != normalizeProjectPath(active) || got.Notes != "legacy active notes" {
		t.Fatalf("legacy active mirror was not preserved: path=%q notes=%q", got.ProjectPath, got.Notes)
	}
	activeID := projectID(active)
	if got.ActiveProjectID != activeID {
		t.Fatalf("active project id=%q want %q", got.ActiveProjectID, activeID)
	}
	var foundHistoric bool
	for _, p := range got.Projects {
		if p.ID == projectID(historic) {
			foundHistoric = true
		}
		if p.ID == activeID && p.Notes != "legacy active notes" {
			t.Fatalf("active project notes=%q", p.Notes)
		}
	}
	if !foundHistoric {
		t.Fatal("historic task project was not imported during v2 migration")
	}
}

func TestVersion3RegistryDoesNotResurrectRemovedHistoricProject(t *testing.T) {
	active := filepath.Join(t.TempDir(), "active")
	removed := filepath.Join(t.TempDir(), "removed")
	now := time.Now().UTC()
	st := State{
		Version:         projectStateVersion,
		Projects:        []Project{{ID: projectID(active), Path: normalizeProjectPath(active), Name: "active", AddedAt: now, LastUsedAt: now}},
		ActiveProjectID: projectID(active),
		ProjectPath:     normalizeProjectPath(active),
		Tasks:           []Task{{ID: "old-task", ProjectPath: removed, CreatedAt: now.Add(-time.Hour)}},
	}
	got := normalizeProjectRegistryState(st)
	if len(got.Projects) != 1 || got.Projects[0].ID != projectID(active) {
		t.Fatalf("v3 normalization resurrected historic project: %#v", got.Projects)
	}
}

func TestEngineMaintainsIndependentProjectNotesAndActiveMirror(t *testing.T) {
	store, err := NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	eng, err := NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	p1 := t.TempDir()
	p2 := t.TempDir()
	first, err := eng.SelectProject(p1)
	if err != nil {
		t.Fatal(err)
	}
	if err := eng.SaveNotes(p1, "notes for one"); err != nil {
		t.Fatal(err)
	}
	second, err := eng.SelectProject(p2)
	if err != nil {
		t.Fatal(err)
	}
	if err := eng.SaveNotes(p2, "notes for two"); err != nil {
		t.Fatal(err)
	}
	if _, err := eng.SelectProject(p1); err != nil {
		t.Fatal(err)
	}
	st := eng.State()
	if st.ActiveProjectID != first.ID || st.ProjectPath != first.Path || st.Notes != "notes for one" {
		t.Fatalf("active compatibility mirror incorrect: %#v", st)
	}
	projects := eng.Projects()
	if len(projects) != 2 {
		t.Fatalf("projects=%d want 2", len(projects))
	}
	var firstNotes, secondNotes string
	for _, p := range projects {
		switch p.ID {
		case first.ID:
			firstNotes = p.Notes
		case second.ID:
			secondNotes = p.Notes
		}
	}
	if firstNotes != "notes for one" || secondNotes != "notes for two" {
		t.Fatalf("per-project notes crossed projects: first=%q second=%q", firstNotes, secondNotes)
	}
}

func TestProjectsSortPinnedBeforeRecentAndRenamePersists(t *testing.T) {
	store, err := NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	eng, err := NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	p1, err := eng.SelectProject(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Millisecond)
	p2, err := eng.SelectProject(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := eng.SetProjectPinned(p1.ID, true); err != nil {
		t.Fatal(err)
	}
	if err := eng.RenameProject(p1.ID, "Pinned Project"); err != nil {
		t.Fatal(err)
	}
	projects := eng.Projects()
	if len(projects) != 2 || projects[0].ID != p1.ID || !projects[0].Pinned || projects[0].Name != "Pinned Project" {
		t.Fatalf("unexpected project order/metadata: %#v", projects)
	}
	if projects[1].ID != p2.ID {
		t.Fatalf("recent unpinned project misplaced: %#v", projects)
	}

	reloaded, err := NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	projects = reloaded.Projects()
	if len(projects) != 2 || projects[0].ID != p1.ID || projects[0].Name != "Pinned Project" || !projects[0].Pinned {
		t.Fatalf("project registry did not persist across engine restart: %#v", projects)
	}
}

func TestTasksForProjectFiltersByRegisteredProject(t *testing.T) {
	p1 := t.TempDir()
	p2 := t.TempDir()
	now := time.Now().UTC()
	st := DefaultState()
	st.Projects = []Project{
		{ID: projectID(p1), Path: normalizeProjectPath(p1), Name: "one", AddedAt: now, LastUsedAt: now},
		{ID: projectID(p2), Path: normalizeProjectPath(p2), Name: "two", AddedAt: now, LastUsedAt: now},
	}
	st.ActiveProjectID = projectID(p1)
	mirrorActiveProject(&st)
	st.Tasks = []Task{
		{ID: "one-a", ProjectPath: p1, Status: TaskCompleted},
		{ID: "two-a", ProjectPath: p2, Status: TaskFailed},
		{ID: "one-b", ProjectPath: p1, Status: TaskRunning},
	}
	store, err := NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(st); err != nil {
		t.Fatal(err)
	}
	eng, err := NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	got := eng.TasksForProject(projectID(p1))
	if len(got) != 2 || got[0].ID != "one-a" || got[1].ID != "one-b" {
		t.Fatalf("project task filter=%#v", got)
	}
	if other := eng.TasksForProject("missing"); other != nil {
		t.Fatalf("missing project returned tasks: %#v", other)
	}
}

func TestRemoveProjectStaysRemovedEvenWithHistoricTasks(t *testing.T) {
	store, err := NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	eng, err := NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	p1, err := eng.SelectProject(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	p2, err := eng.SelectProject(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	eng.mu.Lock()
	eng.state.Tasks = append(eng.state.Tasks, Task{ID: "historic", ProjectPath: p1.Path, CreatedAt: time.Now().Add(-time.Hour)})
	stateWithHistory := cloneState(eng.state)
	eng.mu.Unlock()
	if err := store.Save(stateWithHistory); err != nil {
		t.Fatal(err)
	}
	if err := eng.RemoveProject(p1.ID); err != nil {
		t.Fatal(err)
	}
	for _, p := range eng.Projects() {
		if p.ID == p1.ID {
			t.Fatal("removed project remained in live registry")
		}
	}
	if active, ok := eng.ActiveProject(); !ok || active.ID != p2.ID {
		t.Fatalf("unexpected active project after removal: %#v ok=%t", active, ok)
	}
	reloaded, err := NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range reloaded.Projects() {
		if p.ID == p1.ID {
			t.Fatal("historic task resurrected removed project after reload")
		}
	}
}

func TestStateCloneDoesNotAliasProjectRegistry(t *testing.T) {
	store, err := NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	eng, err := NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	project, err := eng.SelectProject(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	copyState := eng.State()
	if len(copyState.Projects) != 1 {
		t.Fatalf("projects=%d", len(copyState.Projects))
	}
	copyState.Projects[0].Name = "mutated outside engine"
	active, ok := eng.ActiveProject()
	if !ok || active.ID != project.ID {
		t.Fatalf("active project missing: %#v", active)
	}
	if active.Name == "mutated outside engine" {
		t.Fatal("State() returned an aliased project registry slice")
	}
}
