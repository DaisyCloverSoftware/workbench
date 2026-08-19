package mcp

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestMachineToolsAreAdvertisedWithCorrectReviewMetadata(t *testing.T) {
	var inspect, mutate map[string]any
	for _, candidate := range toolsList() {
		switch candidate["name"] {
		case "inspect_machine":
			inspect = candidate
		case "run_machine_command":
			mutate = candidate
		}
	}
	if inspect == nil || mutate == nil {
		t.Fatalf("direct machine tools missing: inspect=%v mutate=%v", inspect != nil, mutate != nil)
	}
	inspectAnn := inspect["annotations"].(map[string]any)
	if inspectAnn["readOnlyHint"] != true || inspectAnn["destructiveHint"] != false || inspectAnn["openWorldHint"] != true {
		t.Fatalf("inspect_machine annotations=%v", inspectAnn)
	}
	mutateAnn := mutate["annotations"].(map[string]any)
	if mutateAnn["readOnlyHint"] != false || mutateAnn["destructiveHint"] != true || mutateAnn["openWorldHint"] != true {
		t.Fatalf("run_machine_command annotations=%v", mutateAnn)
	}
}

func TestMachineToolCallRejectsShellBeforeExecution(t *testing.T) {
	store, err := core.NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	eng, err := core.NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	s := New(eng, 0, "")
	result, ok := s.callTool(context.Background(), "inspect_machine", map[string]any{
		"program": "bash",
		"args":    []any{"-lc", "kubectl get pods"},
	}).(map[string]any)
	if !ok {
		t.Fatalf("unexpected result type %T", result)
	}
	if result["isError"] != true {
		t.Fatalf("shell request should be MCP error: %#v", result)
	}
	structured := result["structuredContent"].(map[string]any)
	if !strings.Contains(strings.ToLower(structured["error"].(string)), "allowlisted") {
		t.Fatalf("unexpected shell rejection: %#v", structured)
	}
}

func TestWorkspaceAdvertisesDirectMachineExecution(t *testing.T) {
	store, err := core.NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	eng, err := core.NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	s := New(eng, 0, "")
	result := s.callTool(context.Background(), "get_workspace", map[string]any{}).(map[string]any)
	structured := result["structuredContent"].(map[string]any)
	if structured["direct_machine_execution"] != true {
		t.Fatalf("direct machine capability missing: %#v", structured)
	}
}
