package core

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRunnerToolListsOnlyRepositoryRoots(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WORKBENCH_RUNNER_ROOT", root)
	repo := filepath.Join(root, "garage")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "init", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	if err := os.MkdirAll(filepath.Join(root, "not-a-repo"), 0o755); err != nil {
		t.Fatal(err)
	}

	response, err := ApplyRunnerToolRequest(context.Background(), RunnerToolRequest{Action: "list_projects"})
	if err != nil {
		t.Fatal(err)
	}
	if !response.OK || len(response.Projects) != 1 {
		t.Fatalf("unexpected response: %+v", response)
	}
	if response.Projects[0].Name != "garage" || response.Projects[0].Ref != "runner://garage" {
		t.Fatalf("unexpected project: %+v", response.Projects[0])
	}
}

func TestSafeRunnerProviderInventoryContainsOnlyCodingWorkers(t *testing.T) {
	providers := []Provider{
		{ID: "chatgpt", Name: "Chat bridge", Installed: true, Authenticated: true},
		{ID: "workbench-runner", Name: "Nested runner", Command: "ssh", Installed: true, Authenticated: true, CanWrite: true, CanRunTools: true},
		{ID: "claude", Name: "Claude", Capability: "coding", Installed: false, CanWrite: true, CanRunTools: true, Cost: CostIncluded, Priority: 40, Status: "not detected"},
		{ID: "copilot", Name: "Copilot", Capability: "coding", Command: "copilot", Installed: true, Authenticated: true, CanWrite: true, CanRunTools: true, Cost: CostIncluded, Priority: 30, Status: "connected"},
	}
	got := safeRunnerProviderInventory(providers)
	if len(got) != 2 {
		t.Fatalf("provider inventory=%+v", got)
	}
	if got[0].ID != "copilot" || !got[0].Ready {
		t.Fatalf("ready provider=%+v", got[0])
	}
	if got[1].ID != "claude" || got[1].Ready || got[1].Installed {
		t.Fatalf("unavailable provider=%+v", got[1])
	}
}
