package mcp

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestSaveNoteUsesTargetProjectNotesWithoutStealingFocus(t *testing.T) {
	store, err := core.NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	eng, err := core.NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	alpha, err := eng.SelectProject(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := eng.SaveNotes(alpha.Path, "alpha-only context"); err != nil {
		t.Fatal(err)
	}
	beta, err := eng.SelectProject(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := eng.SaveNotes(beta.Path, "beta-only context"); err != nil {
		t.Fatal(err)
	}
	if _, err := eng.SelectProject(alpha.Path); err != nil {
		t.Fatal(err)
	}

	server := New(eng, 0, "")
	response, ok := server.callTool(context.Background(), "save_note", map[string]any{
		"project_path": beta.Path,
		"note":         "beta follow-up",
	}).(map[string]any)
	if !ok || response["isError"] == true {
		t.Fatalf("save_note failed: %#v", response)
	}

	active, ok := eng.ActiveProject()
	if !ok || active.ID != alpha.ID {
		t.Fatalf("saving a background project note stole desktop focus: %#v", active)
	}
	alphaAfter, ok := eng.ProjectByPath(alpha.Path)
	if !ok || alphaAfter.Notes != "alpha-only context" {
		t.Fatalf("alpha notes changed unexpectedly: %#v", alphaAfter)
	}
	betaAfter, ok := eng.ProjectByPath(beta.Path)
	if !ok {
		t.Fatal("beta project disappeared")
	}
	if !strings.Contains(betaAfter.Notes, "beta-only context") || !strings.Contains(betaAfter.Notes, "beta follow-up") {
		t.Fatalf("beta notes did not append their own context: %q", betaAfter.Notes)
	}
	if strings.Contains(betaAfter.Notes, "alpha-only context") {
		t.Fatalf("alpha notes leaked into beta: %q", betaAfter.Notes)
	}
}
