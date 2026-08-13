package core

import (
	"strings"
	"testing"
)

func secretLikeFixture() string {
	return "api" + "_key=" + strings.Repeat("x", 24)
}

func TestKnowledgeRejectsSecretLikeSourceAndTag(t *testing.T) {
	for _, tc := range []KnowledgeItem{
		{Scope: ScopeGlobal, Kind: KindFact, Title: "source case", Content: "ordinary content", Source: secretLikeFixture()},
		{Scope: ScopeGlobal, Kind: KindFact, Title: "tag case", Content: "ordinary content", Tags: []string{secretLikeFixture()}},
	} {
		isolateKnowledgeConfig(t)
		if _, err := SaveKnowledge(tc); err == nil {
			t.Fatalf("expected secret-like durable metadata to be rejected: %#v", tc)
		}
	}
}

func TestContextRejectsSecretLikeReferencesAndOpenThreads(t *testing.T) {
	for _, capsule := range []ContextCapsule{
		{Project: "alpha", Objective: "Continue work", State: "ordinary state", References: []string{secretLikeFixture()}},
		{Project: "alpha", Objective: "Continue work", State: "ordinary state", OpenThreads: []string{secretLikeFixture()}},
	} {
		isolateKnowledgeConfig(t)
		if _, err := SaveContextCapsule(capsule); err == nil {
			t.Fatalf("expected secret-like context metadata to be rejected: %#v", capsule)
		}
	}
}
