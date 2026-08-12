package mcp

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestPluginToolsHaveReviewMetadataAndObjectOutputs(t *testing.T) {
	tools := toolsList()
	if len(tools) == 0 {
		t.Fatal("no tools")
	}
	seenWorkspace := false
	for _, tool := range tools {
		name, _ := tool["name"].(string)
		if name == "get_workspace" {
			seenWorkspace = true
		}
		if title, _ := tool["title"].(string); title == "" {
			t.Fatalf("tool %s missing title", name)
		}
		ann, ok := tool["annotations"].(map[string]any)
		if !ok {
			t.Fatalf("tool %s missing annotations", name)
		}
		for _, key := range []string{"readOnlyHint", "destructiveHint", "openWorldHint"} {
			if _, ok := ann[key].(bool); !ok {
				t.Fatalf("tool %s annotation %s missing/bad", name, key)
			}
		}
		out, ok := tool["outputSchema"].(map[string]any)
		if !ok || out["type"] != "object" {
			t.Fatalf("tool %s output schema must be object-shaped: %#v", name, out)
		}
	}
	if !seenWorkspace {
		t.Fatal("get_workspace tool missing")
	}
}

func TestGetWorkspaceUsesServerDefaultProject(t *testing.T) {
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
	got, ok := s.callTool(context.Background(), "get_workspace", map[string]any{}).(map[string]any)
	if !ok {
		t.Fatalf("unexpected result type")
	}
	structured, ok := got["structuredContent"].(map[string]any)
	if !ok {
		t.Fatalf("structuredContent must be an object: %#v", got)
	}
	if structured["project_path"] != project {
		t.Fatalf("project_path=%v want %s", structured["project_path"], project)
	}
	if structured["avoid_work_usage"] != true {
		t.Fatalf("expected scarce Work protection enabled by default")
	}
}
