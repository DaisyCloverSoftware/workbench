package core

import (
	"strings"
	"testing"
)

func TestKnowledgeSecretCheckIncludesMetadataFields(t *testing.T) {
	isolateKnowledgeConfig(t)
	marker := "sk-" + strings.Repeat("x", 24)
	for _, item := range []KnowledgeItem{
		{Scope: ScopeGlobal, Kind: KindFact, Title: "safe title", Content: "safe content", Tags: []string{marker}},
		{Scope: ScopeGlobal, Kind: KindFact, Title: "safe title", Content: "safe content", Source: marker},
	} {
		if _, err := SaveKnowledge(item); err == nil {
			t.Fatal("expected metadata field to be rejected")
		}
	}
}

func TestContextSecretCheckIncludesContinuationFields(t *testing.T) {
	isolateKnowledgeConfig(t)
	marker := "sk-" + strings.Repeat("x", 24)
	for _, capsule := range []ContextCapsule{
		{Project: "alpha", Objective: "continue work", State: "tests pass", References: []string{marker}},
		{Project: "alpha", Objective: "continue work", State: "tests pass", OpenThreads: []string{marker}},
	} {
		if _, err := SaveContextCapsule(capsule); err == nil {
			t.Fatal("expected continuation field to be rejected")
		}
	}
}
