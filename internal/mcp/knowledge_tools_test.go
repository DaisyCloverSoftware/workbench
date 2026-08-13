package mcp

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestMCPMemoryAndContextTools(t *testing.T) {
	config := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", config)
	t.Setenv("APPDATA", config)
	t.Setenv("HOME", config)

	statePath := filepath.Join(t.TempDir(), "state.json")
	store, err := core.NewStoreAt(statePath)
	if err != nil {
		t.Fatal(err)
	}
	eng, err := core.NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(t.TempDir(), "project")
	if err := eng.SaveNotes(project, ""); err != nil {
		t.Fatal(err)
	}
	s := New(eng, 0, "")

	saved := s.callTool(context.Background(), "save_memory", map[string]any{
		"scope":   "project",
		"kind":    "routine",
		"title":   "Go verification",
		"content": "Run gofmt and go test before declaring a Go change complete.",
		"tags":    []any{"go", "verification"},
	}).(map[string]any)
	if saved["isError"] == true {
		t.Fatalf("save_memory failed: %#v", saved)
	}

	found := s.callTool(context.Background(), "search_memory", map[string]any{"query": "Go verification"}).(map[string]any)
	if found["isError"] == true {
		t.Fatalf("search_memory failed: %#v", found)
	}
	structured := found["structuredContent"].(map[string]any)
	if structured["count"].(int) < 1 {
		t.Fatalf("expected saved routine in search: %#v", structured)
	}

	capsule := s.callTool(context.Background(), "save_context", map[string]any{
		"objective":   "Finish memory integration",
		"state":       "Persistence and MCP retrieval are implemented.",
		"decisions":   []any{"Keep project and global scopes distinct."},
		"next_action": "Verify CI.",
	}).(map[string]any)
	if capsule["isError"] == true {
		t.Fatalf("save_context failed: %#v", capsule)
	}

	got := s.callTool(context.Background(), "get_context", map[string]any{}).(map[string]any)
	if got["isError"] == true {
		t.Fatalf("get_context failed: %#v", got)
	}
	gotStructured := got["structuredContent"].(map[string]any)
	if gotStructured["found"] != true {
		t.Fatalf("context capsule not found: %#v", gotStructured)
	}
}
