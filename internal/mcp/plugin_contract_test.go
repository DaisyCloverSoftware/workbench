package mcp

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestPluginToolsHaveReviewMetadataAndObjectOutputs(t *testing.T) {
	tools := toolsList()
	if len(tools) == 0 {
		t.Fatal("no tools")
	}
	seenWorkspace := false
	seenDelegateOperation := false
	for _, tool := range tools {
		name, _ := tool["name"].(string)
		if name == "get_workspace" {
			seenWorkspace = true
		}
		if name == "delegate_operation" {
			seenDelegateOperation = true
			title, _ := tool["title"].(string)
			description, _ := tool["description"].(string)
			lowDescription := strings.ToLower(description)
			if !strings.Contains(strings.ToLower(title), "owner-authorized openclaw") {
				t.Fatalf("delegate_operation title does not identify explicit owner authorization: %q", title)
			}
			for _, required := range []string{
				"owner explicitly asked for openclaw by name",
				"direct-capability or allowlist miss is not authorization",
				"[workbench:openclaw-owner-authorized]",
				"[workbench:operations] alone is routing metadata",
			} {
				if !strings.Contains(lowDescription, strings.ToLower(required)) {
					t.Fatalf("delegate_operation description missing %q: %q", required, description)
				}
			}
			if strings.Contains(lowDescription, "fallback") {
				t.Fatalf("delegate_operation must not advertise OpenClaw as a fallback: %q", description)
			}
			input, ok := tool["inputSchema"].(map[string]any)
			if !ok {
				t.Fatalf("delegate_operation input schema missing: %#v", tool["inputSchema"])
			}
			props, ok := input["properties"].(map[string]any)
			if !ok {
				t.Fatalf("delegate_operation properties missing: %#v", input)
			}
			intentProp, ok := props["intent"].(map[string]any)
			if !ok {
				t.Fatalf("delegate_operation intent property missing: %#v", props)
			}
			intentDescription, _ := intentProp["description"].(string)
			if !strings.Contains(intentDescription, core.OpenClawExplicitAuthorizationPrefix) {
				t.Fatalf("delegate_operation intent does not require explicit authorization marker: %q", intentDescription)
			}
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
	if !seenDelegateOperation {
		t.Fatal("deliberate explicit-use delegate_operation tool missing")
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
	if structured["openclaw_policy"] != "explicit_owner_request_only" {
		t.Fatalf("openclaw_policy=%v, want explicit_owner_request_only", structured["openclaw_policy"])
	}
}
