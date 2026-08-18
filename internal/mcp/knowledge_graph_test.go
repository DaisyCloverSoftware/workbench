package mcp

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func isolateMCPKnowledge(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)
	t.Setenv("APPDATA", root)
	t.Setenv("HOME", root)
}

func TestKnowledgeDiscoveryToolsAreAdvertisedReadOnly(t *testing.T) {
	wanted := map[string]bool{"search_decisions": false, "get_knowledge_graph": false}
	for _, candidate := range toolsList() {
		name, _ := candidate["name"].(string)
		if _, ok := wanted[name]; !ok {
			continue
		}
		annotations, _ := candidate["annotations"].(map[string]any)
		if annotations["readOnlyHint"] != true || annotations["destructiveHint"] != false || annotations["openWorldHint"] != false {
			t.Fatalf("knowledge tool %q annotations=%+v", name, annotations)
		}
		wanted[name] = true
	}
	for name, found := range wanted {
		if !found {
			t.Fatalf("MCP tools/list missing %q", name)
		}
	}
}

func TestKnowledgeDiscoveryToolsReturnStructuredDecisionAndGraphResults(t *testing.T) {
	isolateMCPKnowledge(t)
	project := filepath.Join(t.TempDir(), "project")
	if _, err := core.SaveKnowledge(core.KnowledgeItem{Scope: core.ScopeProject, Project: project, Kind: core.KindDecision, Title: "Routing", Content: "Use included workers first.", Tags: []string{"routing"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := core.SaveContextCapsule(core.ContextCapsule{Project: project, Objective: "Continue routing", State: "Working.", Decisions: []string{"Keep metered fallback opt-in."}}); err != nil {
		t.Fatal(err)
	}
	store, err := core.NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	eng, err := core.NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	s := New(eng, 0, "")

	decisionResult, ok := s.callTool(context.Background(), "search_decisions", map[string]any{"project_path": project, "query": "metered", "limit": 10}).(map[string]any)
	if !ok || decisionResult["isError"] == true {
		t.Fatalf("search_decisions result=%+v", decisionResult)
	}
	decisionStructured, _ := decisionResult["structuredContent"].(map[string]any)
	decisions, _ := decisionStructured["decisions"].([]core.DecisionRecord)
	if len(decisions) != 1 || decisions[0].Decision != "Keep metered fallback opt-in." {
		t.Fatalf("decision structured content=%+v", decisionStructured)
	}

	graphResult, ok := s.callTool(context.Background(), "get_knowledge_graph", map[string]any{"project_path": project, "limit": 20}).(map[string]any)
	if !ok || graphResult["isError"] == true {
		t.Fatalf("get_knowledge_graph result=%+v", graphResult)
	}
	graphStructured, _ := graphResult["structuredContent"].(map[string]any)
	graph, ok := graphStructured["graph"].(core.KnowledgeGraph)
	if !ok || len(graph.Nodes) == 0 || len(graph.Edges) == 0 {
		t.Fatalf("graph structured content=%+v", graphStructured)
	}
	if graphStructured["node_count"] != len(graph.Nodes) || graphStructured["edge_count"] != len(graph.Edges) {
		t.Fatalf("graph counts do not match graph: %+v", graphStructured)
	}
}
