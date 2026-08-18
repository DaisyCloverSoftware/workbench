package main

import (
	"strings"
	"testing"
)

func TestPrivateChatGuideExplainsDecisionSearchAndDerivedKnowledgeGraph(t *testing.T) {
	guide := string(privateChatGuide)
	for _, want := range []string{
		"search_decisions",
		"get_knowledge_graph",
		"derived view, not a second database",
		"Project nodes use opaque identifiers",
	} {
		if !strings.Contains(guide, want) {
			t.Fatalf("private ChatGPT guide missing knowledge discovery rule %q", want)
		}
	}
}
