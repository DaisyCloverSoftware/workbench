package core

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSearchKnowledgeKindsFiltersWithoutCrossProjectLeakage(t *testing.T) {
	isolateKnowledgeConfig(t)
	decision, err := SaveKnowledge(KnowledgeItem{Scope: ScopeProject, Project: "alpha", Kind: KindDecision, Title: "Storage decision", Content: "Use SQLite for durable local state.", Tags: []string{"storage", "state"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := SaveKnowledge(KnowledgeItem{Scope: ScopeProject, Project: "alpha", Kind: KindPattern, Title: "Storage pattern", Content: "Keep writes atomic.", Tags: []string{"storage"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := SaveKnowledge(KnowledgeItem{Scope: ScopeProject, Project: "beta", Kind: KindDecision, Title: "Other storage", Content: "Use a different database."}); err != nil {
		t.Fatal(err)
	}

	items, err := SearchKnowledgeKinds("alpha", "storage", []KnowledgeKind{KindDecision}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != decision.ID || items[0].Kind != KindDecision {
		t.Fatalf("decision-filtered search=%+v", items)
	}
}

func TestSearchDecisionsIncludesDurableMemoryAndContextDecisions(t *testing.T) {
	isolateKnowledgeConfig(t)
	memory, err := SaveKnowledge(KnowledgeItem{Scope: ScopeProject, Project: "alpha", Kind: KindDecision, Title: "Release policy", Content: "Require exact-head CI before merge.", Tags: []string{"release", "ci"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := SaveContextCapsule(ContextCapsule{Project: "alpha", Objective: "Ship safely", State: "Release is pending.", Decisions: []string{"Keep metered fallback opt-in.", "Require exact-head CI before merge."}}); err != nil {
		t.Fatal(err)
	}
	if _, err := SaveContextCapsule(ContextCapsule{Project: "beta", Objective: "Other project", State: "Independent.", Decisions: []string{"Use a beta-only decision."}}); err != nil {
		t.Fatal(err)
	}

	decisions, err := SearchDecisions("alpha", "", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(decisions) != 2 {
		t.Fatalf("decisions=%+v want two de-duplicated alpha decisions", decisions)
	}
	seen := map[string]DecisionRecord{}
	for _, decision := range decisions {
		seen[decision.Decision] = decision
		if strings.Contains(decision.Decision, "beta-only") {
			t.Fatal("decision from another project leaked into alpha search")
		}
	}
	if got := seen["Require exact-head CI before merge."]; got.ID != memory.ID || got.SourceType != "memory" {
		t.Fatalf("explicit durable decision should represent duplicate context decision: %+v", got)
	}
	if got := seen["Keep metered fallback opt-in."]; got.SourceType != "context" || !strings.HasPrefix(got.ID, "ctxdec-") || got.ContextID == "" {
		t.Fatalf("context-only decision missing stable searchable projection: %+v", got)
	}

	filtered, err := SearchDecisions("alpha", "metered", 10)
	if err != nil || len(filtered) != 1 || filtered[0].Decision != "Keep metered fallback opt-in." {
		t.Fatalf("filtered decisions=%+v err=%v", filtered, err)
	}
}

func TestKnowledgeGraphConnectsProjectGlobalMemoryTagsAndContextDecisions(t *testing.T) {
	isolateKnowledgeConfig(t)
	project := `C:\workbench\projects\alpha`
	decision, err := SaveKnowledge(KnowledgeItem{Scope: ScopeProject, Project: project, Kind: KindDecision, Title: "Routing policy", Content: "Use included workers before scarce workers.", Tags: []string{"routing", "cost"}})
	if err != nil {
		t.Fatal(err)
	}
	global, err := SaveKnowledge(KnowledgeItem{Scope: ScopeGlobal, Kind: KindPattern, Title: "Safe retry", Content: "Retry idempotent operations with bounded backoff.", Tags: []string{"retry"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := SaveContextCapsule(ContextCapsule{Project: project, Objective: "Continue routing work", State: "Core path works.", Decisions: []string{"Keep provider routing fail closed."}}); err != nil {
		t.Fatal(err)
	}

	graph, err := BuildKnowledgeGraph(project, "", 20)
	if err != nil {
		t.Fatal(err)
	}
	if !graph.ProjectSelected || len(graph.Nodes) < 6 || len(graph.Edges) < 5 {
		t.Fatalf("knowledge graph too small/incomplete: %+v", graph)
	}
	nodes := map[string]KnowledgeGraphNode{}
	for _, node := range graph.Nodes {
		nodes[node.ID] = node
	}
	if nodes[decision.ID].Type != "memory" || nodes[global.ID].Type != "memory" {
		t.Fatalf("durable memories missing from graph: %+v", graph.Nodes)
	}
	contextDecisionFound := false
	tagFound := false
	for _, node := range graph.Nodes {
		if node.Type == "decision" && node.Content == "Keep provider routing fail closed." {
			contextDecisionFound = true
		}
		if node.Type == "tag" && node.Label == "routing" {
			tagFound = true
		}
	}
	if !contextDecisionFound || !tagFound {
		t.Fatalf("derived graph nodes missing: %+v", graph.Nodes)
	}

	b, err := json.Marshal(graph)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), project) {
		t.Fatalf("knowledge graph leaked host-specific project path: %s", b)
	}
	if !strings.Contains(string(b), "project-knowledge-") || !strings.Contains(string(b), `"relation":"tagged"`) {
		t.Fatalf("graph missing opaque project/tag traversal: %s", b)
	}
}
