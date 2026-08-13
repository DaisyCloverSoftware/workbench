package core

import (
	"strings"
	"testing"
)

func TestKnowledgeRejectsOversizedContent(t *testing.T) {
	isolateKnowledgeConfig(t)
	_, err := SaveKnowledge(KnowledgeItem{
		Scope: ScopeGlobal,
		Kind: KindFact,
		Title: "bounded",
		Content: strings.Repeat("x", knowledgeContentLimit+1),
	})
	if err == nil {
		t.Fatal("expected oversized knowledge content to be rejected")
	}
}

func TestKnowledgeRejectsTooManyTags(t *testing.T) {
	isolateKnowledgeConfig(t)
	tags := make([]string, knowledgeTagCount+1)
	for i := range tags {
		tags[i] = strings.Repeat(string(rune('a'+i)), 2)
	}
	_, err := SaveKnowledge(KnowledgeItem{Scope: ScopeGlobal, Kind: KindFact, Title: "bounded", Content: "ordinary content", Tags: tags})
	if err == nil {
		t.Fatal("expected excessive tags to be rejected")
	}
}

func TestContextRejectsOversizedStateAndLists(t *testing.T) {
	isolateKnowledgeConfig(t)
	if _, err := SaveContextCapsule(ContextCapsule{Project: "alpha", Objective: "continue", State: strings.Repeat("x", contextStateLimit+1)}); err == nil {
		t.Fatal("expected oversized context state to be rejected")
	}
	if _, err := SaveContextCapsule(ContextCapsule{Project: "alpha", Objective: "continue", State: "verified", Decisions: make([]string, contextListCount+1)}); err == nil {
		t.Fatal("expected oversized context list to be rejected")
	}
}

func TestKnowledgeAtContentLimitIsAccepted(t *testing.T) {
	isolateKnowledgeConfig(t)
	_, err := SaveKnowledge(KnowledgeItem{Scope: ScopeGlobal, Kind: KindFact, Title: "bounded", Content: strings.Repeat("x", knowledgeContentLimit)})
	if err != nil {
		t.Fatalf("value at limit was rejected: %v", err)
	}
}
