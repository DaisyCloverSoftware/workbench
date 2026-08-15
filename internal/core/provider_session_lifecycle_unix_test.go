//go:build !windows

package core

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSuccessfulIsolatedTaskClearsAllProviderSessionPointers(t *testing.T) {
	isolateProviderSessions(t)
	isolateKnowledgeConfig(t)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	repo := initPrepareTestRepo(t)
	worker := filepath.Join(t.TempDir(), "fake-copilot")
	if err := os.WriteFile(worker, []byte("#!/bin/sh\nprintf '%s\\n' 'worker completed'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	task := Task{ID: "task-session-cleanup", ProjectPath: repo, Intent: "verify cleanup"}
	if _, err := SaveProviderSession(task.ID, "claude", "550e8400-e29b-41d4-a716-446655440000"); err != nil {
		t.Fatal(err)
	}
	if _, err := SaveProviderSession(task.ID, "codex", "019f244a-489a-7482-803e-1644660fafb7"); err != nil {
		t.Fatal(err)
	}
	provider := Provider{ID: "copilot", Name: "Fake Copilot", Command: worker, Installed: true, Authenticated: true, Cost: CostIncluded, CanWrite: true}
	res, err := RunProviderIsolated(context.Background(), provider, task, Preferences{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Output != "worker completed" {
		t.Fatalf("unexpected worker output: %q", res.Output)
	}
	for _, providerID := range []string{"claude", "codex"} {
		if _, ok, err := ProviderSessionFor(task.ID, providerID); err != nil || ok {
			t.Fatalf("completed task retained %s session: ok=%t err=%v", providerID, ok, err)
		}
	}
}
