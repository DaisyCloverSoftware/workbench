package core

import (
	"strings"
	"testing"
	"time"
)

func TestBuildWorkerPromptWithKnowledgeIncludesReusableMemoryAndCapsule(t *testing.T) {
	task := Task{ProjectPath: "/workspace/project", Intent: "Add retry handling"}
	memories := []KnowledgeItem{
		{ID: "mem-routine", Scope: ScopeGlobal, Kind: KindRoutine, Title: "Retry routine", Content: "Use bounded exponential backoff for idempotent calls."},
		{ID: "mem-code", Scope: ScopeProject, Project: task.ProjectPath, Kind: KindCode, Title: "Existing retry helper", Content: "Reuse internal/retry rather than creating another helper."},
	}
	capsule := ContextCapsule{ID: "ctx-1", Project: task.ProjectPath, Objective: "Finish transport resilience", State: "HTTP client exists and tests pass.", Decisions: []string{"Keep retries bounded."}, NextAction: "Add retry handling.", UpdatedAt: time.Now()}
	prompt := BuildWorkerPromptWithKnowledge(task, memories, &capsule)
	for _, want := range []string{"Relevant Workbench memory", "Retry routine", "Existing retry helper", "Current compact context", "Finish transport resilience", "Reuse before rebuilding"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
}

func TestBuildWorkerPromptWithKnowledgeBoundsMemoryContent(t *testing.T) {
	task := Task{ProjectPath: "/workspace/project", Intent: "Do the thing"}
	memories := []KnowledgeItem{{ID: "mem-large", Scope: ScopeGlobal, Kind: KindCode, Title: "Large", Content: strings.Repeat("x", 50000)}}
	prompt := BuildWorkerPromptWithKnowledge(task, memories, nil)
	if len(prompt) > 30000 {
		t.Fatalf("memory prompt unexpectedly large: %d", len(prompt))
	}
	if !strings.Contains(prompt, "[truncated]") {
		t.Fatal("expected oversized memory to be truncated")
	}
}

func TestBuildWorkerPromptFromStoredKnowledgeLoadsScopedMemoryAndContext(t *testing.T) {
	isolateKnowledgeConfig(t)
	project := "/workspace/project"
	if _, err := SaveKnowledge(KnowledgeItem{Scope: ScopeGlobal, Kind: KindRoutine, Title: "Retry routine", Content: "Use bounded backoff when retrying idempotent requests."}); err != nil {
		t.Fatal(err)
	}
	if _, err := SaveKnowledge(KnowledgeItem{Scope: ScopeProject, Project: project, Kind: KindCode, Title: "Existing helper", Content: "Reuse internal/retry."}); err != nil {
		t.Fatal(err)
	}
	if _, err := SaveContextCapsule(ContextCapsule{Project: project, Objective: "Finish retry support", State: "HTTP client already exists.", NextAction: "Add retry handling."}); err != nil {
		t.Fatal(err)
	}
	prompt := BuildWorkerPromptFromStoredKnowledge(Task{ProjectPath: project, Intent: "Add retry handling"})
	for _, want := range []string{"Retry routine", "Existing helper", "Finish retry support", "Reuse before rebuilding"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("stored worker prompt missing %q:\n%s", want, prompt)
		}
	}
}
