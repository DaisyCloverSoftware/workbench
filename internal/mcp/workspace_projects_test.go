package mcp

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestGetWorkspaceProjectsAreReadOnlyAndPrivacyMinimal(t *testing.T) {
	store, err := core.NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	eng, err := core.NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	p1, err := eng.SelectProject(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := eng.RenameProject(p1.ID, "Pinned Alpha"); err != nil {
		t.Fatal(err)
	}
	if err := eng.SetProjectPinned(p1.ID, true); err != nil {
		t.Fatal(err)
	}
	if err := eng.SaveNotes(p1.Path, "private alpha project notes"); err != nil {
		t.Fatal(err)
	}
	p2, err := eng.SelectProject(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := eng.SaveNotes(p2.Path, "private beta project notes"); err != nil {
		t.Fatal(err)
	}

	st := eng.State()
	st.Tasks = []core.Task{
		{ID: "alpha-ready", ProjectPath: p1.Path, Status: core.TaskCompleted},
		{ID: "alpha-human", ProjectPath: p1.Path, Status: core.TaskNeedsAttention, AttentionQuestion: "Choose A or B"},
		{ID: "beta-running", ProjectPath: p2.Path, Status: core.TaskRunning},
	}
	if err := store.Save(st); err != nil {
		t.Fatal(err)
	}
	eng, err = core.NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	activeBefore, ok := eng.ActiveProject()
	if !ok || activeBefore.ID != p2.ID {
		t.Fatalf("unexpected active project before MCP read: %#v ok=%t", activeBefore, ok)
	}

	server := New(eng, 0, "")
	response, ok := server.callTool(context.Background(), "get_workspace", map[string]any{}).(map[string]any)
	if !ok {
		t.Fatalf("workspace response type=%T", response)
	}
	structured, ok := response["structuredContent"].(map[string]any)
	if !ok {
		t.Fatalf("structured workspace type=%T", response["structuredContent"])
	}
	if structured["active_project_id"] != p2.ID || structured["project_path"] != p2.Path {
		t.Fatalf("active workspace projection=%#v", structured)
	}
	projects, ok := structured["projects"].([]map[string]any)
	if !ok || len(projects) != 2 {
		t.Fatalf("project registry projection=%#v", structured["projects"])
	}

	var alpha, beta map[string]any
	for _, project := range projects {
		switch project["id"] {
		case p1.ID:
			alpha = project
		case p2.ID:
			beta = project
		}
	}
	if alpha == nil || beta == nil {
		t.Fatalf("missing projected projects: %#v", projects)
	}
	if alpha["name"] != "Pinned Alpha" || alpha["pinned"] != true || alpha["active"] != false || alpha["path"] != p1.Path {
		t.Fatalf("alpha projection=%#v", alpha)
	}
	alphaTasks, ok := alpha["tasks"].(map[string]any)
	if !ok || alphaTasks["needs_human"] != 1 || alphaTasks["completed"] != 1 {
		t.Fatalf("alpha task summary=%#v", alpha["tasks"])
	}
	betaTasks, ok := beta["tasks"].(map[string]any)
	if !ok || betaTasks["active"] != 1 || beta["active"] != true {
		t.Fatalf("beta projection=%#v", beta)
	}

	encoded, err := json.Marshal(structured)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.ToLower(string(encoded))
	for _, forbidden := range []string{"private alpha project notes", "private beta project notes", "remote_url", "publication_target", "ciphertext"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("get_workspace leaked private project material %q: %s", forbidden, text)
		}
	}
	activeAfter, ok := eng.ActiveProject()
	if !ok || activeAfter.ID != activeBefore.ID {
		t.Fatalf("read-only workspace projection changed active project: before=%#v after=%#v", activeBefore, activeAfter)
	}
}
