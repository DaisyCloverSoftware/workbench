package core

import "testing"

func TestReusableAssetVersioning(t *testing.T) {
	isolateKnowledgeConfig(t)
	first, err := SaveKnowledge(KnowledgeItem{Scope: ScopeProject, Project: "alpha", Kind: KindRoutine, Title: "Release checks", Content: "Run focused checks."})
	if err != nil { t.Fatal(err) }
	second, err := SaveKnowledge(KnowledgeItem{Scope: ScopeProject, Project: "alpha", Kind: KindRoutine, Title: "Release checks", Content: "Run focused checks and the full suite."})
	if err != nil { t.Fatal(err) }
	if first.AssetVersion != 1 || second.AssetVersion != 2 || second.Supersedes != first.ID { t.Fatalf("unexpected versions: %#v %#v", first, second) }
	items, err := SearchKnowledge("alpha", "release checks", 10)
	if err != nil { t.Fatal(err) }
	if len(items) != 1 || items[0].ID != second.ID { t.Fatalf("expected latest asset only, got %#v", items) }
}
