package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestSelectRunnerRelayActivityPathsKeepsPendingAndRecentPairs(t *testing.T) {
	repo := t.TempDir()
	runRelayGit(t, repo, "init")
	runRelayGit(t, repo, "config", "user.email", "workbench@example.invalid")
	runRelayGit(t, repo, "config", "user.name", "Workbench Test")

	writeRelayTestFile(t, repo, "relay/control/old_done.json", `{"id":"old_done"}`)
	writeRelayTestFile(t, repo, "relay/control-outbox/old_done.json", `{"id":"old_done"}`)
	runRelayGit(t, repo, "add", ".")
	old := exec.Command("git", "-C", repo, "commit", "-m", "old history")
	old.Env = append(os.Environ(), "GIT_AUTHOR_DATE=2026-08-20T10:00:00Z", "GIT_COMMITTER_DATE=2026-08-20T10:00:00Z")
	if out, err := old.CombinedOutput(); err != nil {
		t.Fatalf("old commit: %v\n%s", err, out)
	}

	writeRelayTestFile(t, repo, "relay/control/recent_done.json", `{"id":"recent_done"}`)
	writeRelayTestFile(t, repo, "relay/control-outbox/recent_done.json", `{"id":"recent_done"}`)
	writeRelayTestFile(t, repo, "relay/control/pending_now.json", `{"id":"pending_now"}`)
	runRelayGit(t, repo, "add", ".")
	runRelayGit(t, repo, "commit", "-m", "current activity")
	runRelayGit(t, repo, "update-ref", "refs/remotes/origin/main", "HEAD")

	paths, err := selectRunnerRelayActivityPaths(repo, "origin/main")
	if err != nil {
		t.Fatal(err)
	}
	set := map[string]bool{}
	for _, path := range paths {
		set[path] = true
	}
	for _, want := range []string{
		"relay/control/pending_now.json",
		"relay/control/recent_done.json",
		"relay/control-outbox/recent_done.json",
	} {
		if !set[want] {
			t.Fatalf("missing %s from %#v", want, paths)
		}
	}
	if set["relay/control/old_done.json"] || set["relay/control-outbox/old_done.json"] {
		t.Fatalf("old completed history leaked into bounded inventory: %#v", paths)
	}
}

func TestReadRunnerRelayActivityArchiveParsesBoundedSelection(t *testing.T) {
	repo := t.TempDir()
	runRelayGit(t, repo, "init")
	runRelayGit(t, repo, "config", "user.email", "workbench@example.invalid")
	runRelayGit(t, repo, "config", "user.name", "Workbench Test")
	writeRelayTestFile(t, repo, "relay/control/windows_live.json", `{"version":1,"id":"windows_live","action":"run_windows_unreal_smoke","args":{}}`)
	runRelayGit(t, repo, "add", ".")
	runRelayGit(t, repo, "commit", "-m", "pending windows job")
	runRelayGit(t, repo, "update-ref", "refs/remotes/origin/main", "HEAD")

	raw, err := readRunnerRelayActivityArchive(repo)
	if err != nil {
		t.Fatal(err)
	}
	items, err := parseRunnerChatActivity(raw, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "windows_live" || items[0].State != "running" {
		t.Fatalf("activity=%#v", items)
	}
}

func writeRelayTestFile(t *testing.T, repo, name, content string) {
	t.Helper()
	path := filepath.Join(repo, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func runRelayGit(t *testing.T, repo string, args ...string) {
	t.Helper()
	cmdArgs := append([]string{"-C", repo}, args...)
	if out, err := exec.Command("git", cmdArgs...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
