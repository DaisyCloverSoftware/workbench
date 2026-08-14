package core

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestIsolatedWorkerMemoryStaysScopedToSourceProject(t *testing.T) {
	isolateKnowledgeConfig(t)
	sourceProject := "/workspace/source-project"
	worktree := "/cache/workbench/task-worktree"

	if _, err := SaveKnowledge(KnowledgeItem{
		Scope:   ScopeProject,
		Project: sourceProject,
		Kind:    KindDecision,
		Title:   "Source project decision",
		Content: "Keep durable memory attached to the real repository.",
	}); err != nil {
		t.Fatal(err)
	}

	task := Task{
		ID:                "task-memory-scope",
		ProjectPath:       worktree,
		Intent:            "source project decision",
		memoryProjectPath: sourceProject,
	}
	prompt := BuildWorkerPromptFromStoredKnowledge(task)
	if !strings.Contains(prompt, "Source project decision") {
		t.Fatalf("isolated prompt did not load source-project memory:\n%s", prompt)
	}
	if !strings.Contains(prompt, worktree) {
		t.Fatalf("worker prompt did not retain isolated execution path:\n%s", prompt)
	}

	clean := persistWorkerMemories(task, "done\nWORKBENCH_MEMORY: {\"kind\":\"fact\",\"title\":\"Captured source fact\",\"content\":\"This belongs to the real project.\"}")
	if clean != "done" {
		t.Fatalf("unexpected cleaned worker report: %q", clean)
	}
	items, err := SearchKnowledge(sourceProject, "Captured source fact", 10)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range items {
		if item.Title == "Captured source fact" {
			found = true
			if item.Project != sourceProject {
				t.Fatalf("captured memory project=%q want=%q", item.Project, sourceProject)
			}
		}
	}
	if !found {
		t.Fatal("captured worker memory was not stored under the source project")
	}
	worktreeItems, err := SearchKnowledge(worktree, "Captured source fact", 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range worktreeItems {
		if item.Title == "Captured source fact" && item.Scope == ScopeProject {
			t.Fatalf("captured worker memory leaked into ephemeral worktree scope: %#v", item)
		}
	}
}

func TestIsolatedMemoryProjectIdentityIsNotSerialized(t *testing.T) {
	task := Task{ProjectPath: "/cache/task", memoryProjectPath: "/private/source"}
	b, err := json.Marshal(task)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "/private/source") || strings.Contains(string(b), "memory_project") {
		t.Fatalf("transient memory project leaked into JSON: %s", b)
	}
}
