package core

import (
	"os"
	"path/filepath"
	"testing"
)

func isolateKnowledgeConfig(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)
	t.Setenv("APPDATA", root)
	t.Setenv("HOME", root)
}

func TestKnowledgeProjectAndGlobalScopes(t *testing.T) {
	isolateKnowledgeConfig(t)
	global, err := SaveKnowledge(KnowledgeItem{Scope: ScopeGlobal, Kind: KindPattern, Title: "HTTP retry", Content: "Retry idempotent requests with bounded backoff.", Tags: []string{"http", "retry"}})
	if err != nil {
		t.Fatal(err)
	}
	project, err := SaveKnowledge(KnowledgeItem{Scope: ScopeProject, Project: "alpha", Kind: KindDecision, Title: "Database", Content: "Use SQLite for local state."})
	if err != nil {
		t.Fatal(err)
	}
	items, err := SearchKnowledge("alpha", "database retry", 10)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, item := range items {
		seen[item.ID] = true
	}
	if !seen[global.ID] || !seen[project.ID] {
		t.Fatalf("expected global and project memory, got %#v", items)
	}
	items, err = SearchKnowledge("beta", "database", 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range items {
		if item.ID == project.ID {
			t.Fatal("project memory leaked into another project")
		}
	}
}

func TestKnowledgeDeduplicatesExactReusableItem(t *testing.T) {
	isolateKnowledgeConfig(t)
	first, err := SaveKnowledge(KnowledgeItem{Scope: ScopeGlobal, Kind: KindRoutine, Title: "Go checks", Content: "Run gofmt then go test ./..."})
	if err != nil {
		t.Fatal(err)
	}
	second, err := SaveKnowledge(KnowledgeItem{Scope: ScopeGlobal, Kind: KindRoutine, Title: "Go checks", Content: "Run gofmt then go test ./..."})
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Fatalf("duplicate routine got new id: %s != %s", first.ID, second.ID)
	}
}

func TestKnowledgeRejectsSecretLikeContent(t *testing.T) {
	isolateKnowledgeConfig(t)
	fakeCredential := "sk-" + "abcdefghijklmnopqrstuvwxyz123456"
	_, err := SaveKnowledge(KnowledgeItem{Scope: ScopeGlobal, Kind: KindFact, Title: "credential", Content: fakeCredential})
	if err == nil {
		t.Fatal("expected secret-like knowledge to be rejected")
	}
}

func TestContextCapsuleRoundTrip(t *testing.T) {
	isolateKnowledgeConfig(t)
	capsule, err := SaveContextCapsule(ContextCapsule{Project: "alpha", Objective: "Finish the parser", State: "Lexer is implemented and tests pass.", Decisions: []string{"Keep parser deterministic."}, References: []string{"mem-1"}, NextAction: "Implement AST nodes."})
	if err != nil {
		t.Fatal(err)
	}
	got, ok, err := LatestContextCapsule("alpha")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || got.ID != capsule.ID || got.NextAction != "Implement AST nodes." {
		t.Fatalf("unexpected capsule: %#v", got)
	}
	path, err := KnowledgeStatePath()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "knowledge.json" {
		t.Fatalf("unexpected knowledge path: %s", path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}
