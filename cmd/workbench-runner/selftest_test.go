package main

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestSelftestVerifiesWorkbenchWithoutExternalWorkers(t *testing.T) {
	root := t.TempDir()
	config := t.TempDir()
	cache := t.TempDir()
	t.Setenv("WORKBENCH_RUNNER_ROOT", root)
	t.Setenv("WORKBENCH_KEEP_SELFTEST", "")
	t.Setenv("XDG_CONFIG_HOME", config)
	t.Setenv("APPDATA", config)
	t.Setenv("XDG_CACHE_HOME", cache)
	t.Setenv("LOCALAPPDATA", cache)

	if err := selftest(); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("deterministic selftest left runner-root entries behind: %v", entries)
	}
}

func TestCreateSelftestRepositoryHasCommittedCleanBaseline(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WORKBENCH_RUNNER_ROOT", root)
	t.Setenv("WORKBENCH_KEEP_SELFTEST", "")

	dir, _, err := createSelftestRepository(".workbench-fixture-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	head, err := runSelftestGit(ctx, dir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(head) == "" {
		t.Fatal("selftest repository has no committed HEAD")
	}
	status, err := runSelftestGit(ctx, dir, "status", "--porcelain")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(status) != "" {
		t.Fatalf("selftest repository baseline is dirty: %q", strings.TrimSpace(status))
	}
}

func TestFindLiveSelftestReviewCommitUsesWorkbenchOwnedBranch(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WORKBENCH_RUNNER_ROOT", root)
	dir, _, err := createSelftestRepository(".workbench-fixture-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	head, err := runSelftestGit(ctx, dir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runSelftestGit(ctx, dir, "branch", "workbench/runner-live-selftest-deadbeef", strings.TrimSpace(head)); err != nil {
		t.Fatal(err)
	}
	got, err := findLiveSelftestReviewCommit(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(got) != strings.TrimSpace(head) {
		t.Fatalf("review commit=%q want %q", strings.TrimSpace(got), strings.TrimSpace(head))
	}
}
