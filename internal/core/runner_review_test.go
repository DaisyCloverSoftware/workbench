package core

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunnerReviewRequestCarriesNoPublicationTarget(t *testing.T) {
	req := RunnerReviewRequest{
		Action:  "retry",
		Project: "/project",
		Review: TaskReviewResult{
			Changed:      true,
			BaseRevision: "base",
			Fingerprint:  "fingerprint",
			Branch:       "workbench/task",
			Commit:       "commit",
			Files:        []string{"file.go"},
		},
	}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.ToLower(string(b))
	if strings.Contains(text, "remote_url") || strings.Contains(text, "publication_target") || strings.Contains(text, "github.com/") || strings.Contains(text, "git@") {
		t.Fatalf("runner review request exposed publication authority: %s", text)
	}
}

func TestApplyRunnerReviewRequestRetriesPreparedReviewWithoutAI(t *testing.T) {
	isolateKnowledgeConfig(t)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	root := t.TempDir()
	t.Setenv("WORKBENCH_RUNNER_ROOT", root)
	repo := filepath.Join(root, "project")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "init", "-q", repo)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	prepareTestGit(t, repo, "config", "user.name", "Test")
	prepareTestGit(t, repo, "config", "user.email", "test@example.invalid")
	if err := os.WriteFile(filepath.Join(repo, "tracked.txt"), []byte("before\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	prepareTestGit(t, repo, "add", "tracked.txt")
	prepareTestGit(t, repo, "commit", "-q", "-m", "initial")

	if _, err := SavePublicationPolicy(PublicationPolicy{Project: repo, Mode: PublicationPrepare}); err != nil {
		t.Fatal(err)
	}
	ws, err := CreateTaskWorkspace(context.Background(), repo, "task-runner-review")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws.Workspace, "tracked.txt"), []byte("runner review\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	review, err := FinalizeTaskWorkspace(context.Background(), ws)
	if err != nil {
		t.Fatal(err)
	}

	remote := filepath.Join(t.TempDir(), "remote.git")
	if out, err := exec.Command("git", "init", "--bare", "-q", remote).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %v\n%s", err, out)
	}
	if _, err := SavePublicationPolicy(PublicationPolicy{Project: repo, Mode: PublicationPublish, RemoteURL: remote}); err != nil {
		t.Fatal(err)
	}
	response, err := ApplyRunnerReviewRequest(context.Background(), RunnerReviewRequest{Action: "retry", Project: repo, Review: review})
	if err != nil {
		t.Fatal(err)
	}
	if !response.OK || response.Review == nil || response.Review.PublicationStatus != ReviewPublicationPublished {
		t.Fatalf("unexpected runner review response: %#v", response)
	}
}

func TestRunnerReviewControlsAreNotModelSafeCommands(t *testing.T) {
	for _, command := range []string{
		"workbench-runner review-json",
		"workbench-runner review retry task-1",
	} {
		if IsSafeCommand(command) {
			t.Fatalf("runner review control must remain outside model-safe commands: %q", command)
		}
	}
}
