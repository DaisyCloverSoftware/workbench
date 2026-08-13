package core

import "testing"

func TestReusableAssetVersioning(t *testing.T) {
	isolateKnowledgeConfig(t)
	first, firstMeta, err := SaveReusableAsset(KnowledgeItem{Scope: ScopeProject, Project: "alpha", Kind: KindRoutine, Title: "Release checks", Content: "Run focused checks."}, "")
	if err != nil { t.Fatal(err) }
	second, secondMeta, err := SaveReusableAsset(KnowledgeItem{Scope: ScopeProject, Project: "alpha", Kind: KindRoutine, Title: "Release checks", Content: "Run focused checks and the full suite."}, "full suite passed")
	if err != nil { t.Fatal(err) }
	if firstMeta.Version != 1 || secondMeta.Version != 2 || secondMeta.Supersedes != first.ID { t.Fatalf("unexpected versions: %#v %#v", firstMeta, secondMeta) }
	if !ReusableAssetVerified(second) || secondMeta.VerifiedAt == nil { t.Fatalf("expected verified latest asset: %#v %#v", second, secondMeta) }
	items, err := SearchKnowledge("alpha", "release checks", 10)
	if err != nil { t.Fatal(err) }
	items = FilterActiveReusableKnowledge(items)
	if len(items) != 1 || items[0].ID != second.ID { t.Fatalf("expected latest asset only, got %#v", items) }
}

func TestReusableAssetExactDuplicateKeepsVersion(t *testing.T) {
	isolateKnowledgeConfig(t)
	first, firstMeta, err := SaveReusableAsset(KnowledgeItem{Scope: ScopeGlobal, Kind: KindCode, Title: "Retry helper", Content: "Use bounded backoff."}, "")
	if err != nil { t.Fatal(err) }
	second, secondMeta, err := SaveReusableAsset(KnowledgeItem{Scope: ScopeGlobal, Kind: KindCode, Title: "Retry helper", Content: "Use bounded backoff."}, "")
	if err != nil { t.Fatal(err) }
	if first.ID != second.ID || firstMeta.Version != 1 || secondMeta.Version != 1 { t.Fatalf("duplicate changed version: %#v %#v", firstMeta, secondMeta) }
}
