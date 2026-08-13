//go:build !windows

package core

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestRouterUsesCheaperWorkerBeforeCodex(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture")
	}
	bin := t.TempDir()
	project := t.TempDir()
	writeExe := func(name, body string) {
		p := filepath.Join(bin, name)
		if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeExe("agy", `printf '%s\n' 'completed using included fixture'`)
	writeExe("codex", `if [ "$1" = "login" ]; then exit 0; fi; printf '%s\n' '{"type":"message","text":"codex should not run"}'`)
	old := os.Getenv("PATH")
	t.Setenv("PATH", bin+":"+old)
	store, err := NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	eng, err := NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	task, err := eng.Delegate("test", "make a harmless source change", project)
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		cur, _ := eng.Task(task.ID)
		if cur.Status == TaskCompleted {
			if cur.ProviderID != "antigravity" {
				t.Fatalf("provider=%s, wanted antigravity", cur.ProviderID)
			}
			if cur.ConsumesWork {
				t.Fatal("cheaper route should not consume Work")
			}
			return
		}
		if cur.Status == TaskFailed {
			t.Fatalf("task failed: %s", cur.Error)
		}
		time.Sleep(30 * time.Millisecond)
	}
	t.Fatal("timed out waiting for task")
}

func TestRouteCandidatesSkipKnownUnavailableWorkers(t *testing.T) {
	providers := []Provider{
		{ID: "signed-out", Installed: true, Authenticated: false, CanWrite: true, Command: "worker", Cost: CostIncluded, Priority: 10},
		{ID: "ready", Installed: true, Authenticated: true, CanWrite: true, Command: "worker", Cost: CostIncluded, Priority: 20},
		{ID: "workbench-runner", Installed: true, Authenticated: true, CanWrite: true, Command: "ssh", Cost: CostIncluded, Priority: 5},
	}
	got := routeCandidates(providers, Preferences{}, Task{})
	if len(got) != 1 || got[0].ID != "ready" {
		t.Fatalf("unexpected candidates: %#v", got)
	}
}

func TestRouteCandidatesAllowConfiguredRunnerTransport(t *testing.T) {
	providers := []Provider{{ID: "workbench-runner", Installed: true, Authenticated: true, CanWrite: true, Command: "ssh", Cost: CostIncluded, Priority: 5}}
	got := routeCandidates(providers, Preferences{OpenClawSSHHost: "runner.example.invalid"}, Task{})
	if len(got) != 1 || got[0].ID != "workbench-runner" {
		t.Fatalf("configured runner was not eligible: %#v", got)
	}
}

func TestCloneStateDeepCopiesTaskAttempts(t *testing.T) {
	state := DefaultState()
	state.Tasks = []Task{{ID: "task-one", Attempts: []string{"first"}}}
	clone := cloneState(state)
	clone.Tasks[0].Attempts[0] = "changed"
	clone.Tasks[0].Attempts = append(clone.Tasks[0].Attempts, "second")
	if state.Tasks[0].Attempts[0] != "first" || len(state.Tasks[0].Attempts) != 1 {
		t.Fatalf("clone mutated original attempts: %#v", state.Tasks[0].Attempts)
	}
}
