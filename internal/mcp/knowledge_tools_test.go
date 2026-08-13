package mcp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestKnowledgeToolsRoundTripCompactContext(t *testing.T) {
	store, err := core.NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	eng, err := core.NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(t.TempDir(), "project")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := eng.SaveNotes(project, ""); err != nil {
		t.Fatal(err)
	}
	s := New(eng, 0, "")

	checkpointResult := s.callTool(context.Background(), "save_checkpoint", map[string]any{
		"summary":      "The memory layer is the current implementation slice.",
		"decisions":    []any{"Keep conversation history disposable."},
		"open_loops":   []any{"Add semantic retrieval later."},
		"next_actions": []any{"Use compact context on the next task."},
	})
	assertToolOK(t, checkpointResult)

	memoryResult := s.callTool(context.Background(), "remember", map[string]any{
		"scope":   "project",
		"kind":    "constraint",
		"title":   "Compact active context",
		"summary": "Do not require a complete old transcript to resume work.",
		"tags":    []any{"context", "resume"},
	})
	assertToolOK(t, memoryResult)

	routineResult := s.callTool(context.Background(), "save_routine", map[string]any{
		"scope":       "global",
		"name":        "Checkpoint before compaction",
		"description": "Persist decisions and open loops before a long conversation is replaced.",
		"triggers":    []any{"context limit", "new conversation"},
		"steps":       []any{"Summarise current state.", "Save durable decisions.", "Record next actions."},
		"tags":        []any{"memory", "compaction"},
	})
	assertToolOK(t, routineResult)

	result := s.callTool(context.Background(), "get_context_pack", map[string]any{"query": "context compaction resume"})
	got, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("unexpected context pack result type: %#v", result)
	}
	structured, ok := got["structuredContent"].(core.ContextPack)
	if !ok {
		t.Fatalf("structuredContent is not a ContextPack: %#v", got["structuredContent"])
	}
	for _, want := range []string{"CURRENT CHECKPOINT", "Compact active context", "Checkpoint before compaction"} {
		if !strings.Contains(structured.ContextText, want) {
			t.Fatalf("context pack missing %q:\n%s", want, structured.ContextText)
		}
	}
}

func TestKnowledgeToolsAdvertised(t *testing.T) {
	want := map[string]bool{
		"get_context_pack": false,
		"recall_memory":    false,
		"find_routines":    false,
		"remember":         false,
		"save_checkpoint":  false,
		"save_routine":     false,
	}
	for _, item := range toolsList() {
		name, _ := item["name"].(string)
		if _, ok := want[name]; ok {
			want[name] = true
		}
	}
	for name, seen := range want {
		if !seen {
			t.Fatalf("knowledge tool %s is not advertised", name)
		}
	}
}

func assertToolOK(t *testing.T, result any) {
	t.Helper()
	got, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("unexpected tool result: %#v", result)
	}
	if isErr, _ := got["isError"].(bool); isErr {
		t.Fatalf("tool returned error: %#v", got)
	}
}
