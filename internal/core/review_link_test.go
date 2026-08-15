package core

import (
	"testing"
)

func TestReviewPullRequestURLUsesPrivatePublicationPolicy(t *testing.T) {
	isolateKnowledgeConfig(t)
	repo := initPrepareTestRepo(t)
	if _, err := SavePublicationPolicy(PublicationPolicy{
		Project:   repo,
		Mode:      PublicationPublish,
		RemoteURL: "https://github.com/DaisyCloverSoftware/workbench.git",
	}); err != nil {
		t.Fatal(err)
	}
	task := Task{
		ProjectPath: repo,
		Review: &TaskReviewResult{
			Changed:           true,
			PublicationStatus: ReviewPublicationPublished,
			PullRequestStatus: ReviewPullRequestAvailable,
			PullRequestNumber: 42,
			PullRequestState:  "open",
		},
	}
	got := ReviewPullRequestURL(task)
	want := "https://github.com/DaisyCloverSoftware/workbench/pull/42"
	if got != want {
		t.Fatalf("review URL=%q want %q", got, want)
	}
}

func TestReviewPullRequestURLRequiresAvailableStructuredPR(t *testing.T) {
	if got := ReviewPullRequestURL(Task{}); got != "" {
		t.Fatalf("unexpected review URL without review metadata: %q", got)
	}
}
