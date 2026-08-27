//go:build !windows

package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestRelayGitDeadlineKillsHookDescendants(t *testing.T) {
	repo := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	git("init", "--quiet")
	git("config", "user.email", "workbench-test@example.invalid")
	git("config", "user.name", "Workbench Test")

	tracked := filepath.Join(repo, "tracked.txt")
	if err := os.WriteFile(tracked, []byte("initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git("add", "tracked.txt")
	git("commit", "--quiet", "-m", "initial")
	if err := os.WriteFile(tracked, []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git("add", "tracked.txt")

	marker := filepath.Join(t.TempDir(), "escaped-hook-marker")
	t.Setenv("WORKBENCH_RELAY_TEST_MARKER", marker)
	hook := filepath.Join(repo, ".git", "hooks", "pre-commit")
	hookBody := "#!/bin/sh\n(sleep 1; printf survived > \"$WORKBENCH_RELAY_TEST_MARKER\") &\nwait\n"
	if err := os.WriteFile(hook, []byte(hookBody), 0o755); err != nil {
		t.Fatal(err)
	}

	started := time.Now()
	_, err := relayGitCombinedOutput(context.Background(), 150*time.Millisecond, repo, "commit", "-m", "must-time-out")
	if err == nil {
		t.Fatal("expected bounded git commit to time out")
	}
	if elapsed := time.Since(started); elapsed > 3*time.Second {
		t.Fatalf("bounded git command returned too slowly: %s", elapsed)
	}

	time.Sleep(1200 * time.Millisecond)
	if _, statErr := os.Stat(marker); statErr == nil {
		t.Fatal("git hook descendant survived command deadline")
	} else if !errors.Is(statErr, os.ErrNotExist) {
		t.Fatal(statErr)
	}
}
