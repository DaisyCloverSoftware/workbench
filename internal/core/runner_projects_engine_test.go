package core

import (
	"path/filepath"
	"testing"
)

func TestCanonicalProjectSelectionAcceptsRunnerReference(t *testing.T) {
	got, err := canonicalProjectSelection("runner://garage")
	if err != nil {
		t.Fatal(err)
	}
	if got != "runner://garage" {
		t.Fatalf("unexpected runner project %q", got)
	}
}

func TestRegisterRunnerProjectsDoesNotRequireLocalDirectory(t *testing.T) {
	store, err := NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	engine, err := NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	added, err := engine.RegisterRunnerProjects([]RunnerProjectInfo{{Name: "garage", Ref: "runner://garage"}})
	if err != nil {
		t.Fatal(err)
	}
	if added != 1 {
		t.Fatalf("expected one added project, got %d", added)
	}
	project, ok := engine.ActiveProject()
	if !ok || project.Path != "runner://garage" || project.Name != "garage" {
		t.Fatalf("unexpected active project: %+v, %v", project, ok)
	}
}
