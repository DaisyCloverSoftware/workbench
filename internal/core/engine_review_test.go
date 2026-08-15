package core

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEngineRetryTaskReviewDeliveryUpdatesCompletedTaskWithoutRecoding(t *testing.T) {
	isolateKnowledgeConfig(t)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	repo := initPrepareTestRepo(t)
	ctx := context.Background()

	if _, err := SavePublicationPolicy(PublicationPolicy{Project: repo, Mode: PublicationPrepare}); err != nil {
		t.Fatal(err)
	}
	ws, err := CreateTaskWorkspace(ctx, repo, "task-engine-review")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws.Workspace, "tracked.txt"), []byte("engine retry\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	review, err := FinalizeTaskWorkspace(ctx, ws)
	if err != nil {
		t.Fatal(err)
	}

	statePath := filepath.Join(t.TempDir(), "state.json")
	store, err := NewStoreAt(statePath)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	state := DefaultState()
	state.Tasks = []Task{{
		ID:          "task-engine-review",
		Title:       "Review delivery",
		Intent:      "already completed code",
		ProjectPath: repo,
		Status:      TaskCompleted,
		ProviderID:  "claude",
		Review:      cloneTaskReviewResult(&review),
		CreatedAt:   now,
		UpdatedAt:   now,
	}}
	if err := store.Save(state); err != nil {
		t.Fatal(err)
	}
	eng, err := NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}

	remote := filepath.Join(t.TempDir(), "review.git")
	if out, err := exec.Command("git", "init", "--bare", "-q", remote).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %v\n%s", err, out)
	}
	if _, err := SavePublicationPolicy(PublicationPolicy{Project: repo, Mode: PublicationPublish, RemoteURL: remote}); err != nil {
		t.Fatal(err)
	}
	if err := eng.RetryTaskReviewDelivery("task-engine-review"); err != nil {
		t.Fatal(err)
	}
	got, ok := eng.Task("task-engine-review")
	if !ok {
		t.Fatal("task disappeared")
	}
	if got.Status != TaskCompleted || got.Review == nil || got.Review.PublicationStatus != ReviewPublicationPublished || !got.Review.Published {
		t.Fatalf("review retry changed task incorrectly: %#v", got)
	}
	if len(got.Attempts) != 1 || !strings.Contains(got.Attempts[0], "review delivery retry") {
		t.Fatalf("retry audit was not recorded: %v", got.Attempts)
	}
	if got.ProviderID != "claude" || got.Intent != "already completed code" {
		t.Fatalf("retry rewrote coding identity: %#v", got)
	}
}
