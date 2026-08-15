package core

import (
	"strings"
	"testing"
)

func TestPresentTaskPrefersGitHubPullRequestAsNextAction(t *testing.T) {
	p := PresentTask(Task{
		Status: TaskCompleted,
		Review: &TaskReviewResult{
			Changed:           true,
			Branch:            "workbench/task-pr",
			Commit:            "abcdef0123456789",
			PublicationStatus: ReviewPublicationPublished,
			Published:         true,
			PullRequestStatus: ReviewPullRequestAvailable,
			PullRequestNumber: 42,
			PullRequestState:  "open",
		},
	})
	if p.PullRequestNumber != 42 || p.PullRequestStatus != ReviewPullRequestAvailable || p.PullRequestState != "open" {
		t.Fatalf("PR metadata missing from presentation: %#v", p)
	}
	if !strings.Contains(p.NextAction, "Review GitHub PR #42") {
		t.Fatalf("completed task did not point user to real review artifact: %q", p.NextAction)
	}
}

func TestPresentTaskPublishedBranchCanRetryPRWithoutRecoding(t *testing.T) {
	p := PresentTask(Task{
		Status: TaskCompleted,
		Review: &TaskReviewResult{
			Changed:           true,
			Branch:            "workbench/task-pr-failed",
			Commit:            "fedcba9876543210",
			PublicationStatus: ReviewPublicationPublished,
			Published:         true,
			PullRequestStatus: ReviewPullRequestUnavailable,
		},
	})
	if !strings.Contains(p.NextAction, "Retry review delivery") || !strings.Contains(p.NextAction, "coding will not run again") {
		t.Fatalf("PR delivery failure did not preserve coding completion: %q", p.NextAction)
	}
}
