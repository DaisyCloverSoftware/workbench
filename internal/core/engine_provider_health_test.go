//go:build !windows

package core

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEngineSkipsCoolingProviderAndUsesNextEligibleWorker(t *testing.T) {
	isolateProviderHealthCache(t)
	bin := t.TempDir()
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(gitPath, filepath.Join(bin, "git")); err != nil {
		t.Fatal(err)
	}
	writeHealthExe := func(name, body string) {
		t.Helper()
		path := filepath.Join(bin, name)
		if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	antigravityMarker := filepath.Join(t.TempDir(), "antigravity-ran")
	writeHealthExe("agy", `printf '%s\n' ran > '`+antigravityMarker+`'; printf '%s\n' 'antigravity should have been skipped'`)
	writeHealthExe("codex", `if [ "$1" = "login" ]; then exit 0; fi; printf '%s\n' '{"type":"message","text":"codex completed after cooldown skip"}'`)
	t.Setenv("PATH", bin)

	now := time.Now().UTC()
	if _, err := recordProviderRetryableFailureAt("antigravity", RunResult{Retryable: true, WorkerUnavailable: "quota exceeded"}, errors.New("quota exceeded"), now); err != nil {
		t.Fatal(err)
	}

	project := initPrepareTestRepo(t)
	store, err := NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	eng, err := NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	task, err := eng.Delegate("test", "verify cooldown routing", project)
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		cur, _ := eng.Task(task.ID)
		switch cur.Status {
		case TaskCompleted:
			if cur.ProviderID != "codex" {
				t.Fatalf("provider=%q want codex; attempts=%#v", cur.ProviderID, cur.Attempts)
			}
			if _, err := os.Stat(antigravityMarker); !os.IsNotExist(err) {
				t.Fatalf("cooling Antigravity was invoked; stat err=%v", err)
			}
			joined := strings.Join(cur.Attempts, "\n")
			if !strings.Contains(joined, "Google Antigravity CLI: skipped until") || !strings.Contains(joined, "quota or rate limit") {
				t.Fatalf("task attempts do not explain cooldown skip: %#v", cur.Attempts)
			}
			waitForPersistedTaskStatus(t, store, task.ID, TaskCompleted, deadline)
			return
		case TaskFailed:
			waitForPersistedTaskStatus(t, store, task.ID, TaskFailed, deadline)
			t.Fatalf("task failed: %s; attempts=%#v", cur.Error, cur.Attempts)
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("timed out waiting for cooldown-routed task")
}
