package core

import "testing"

func TestReviewPullRequestURLUsesRunnerPolicyMirror(t *testing.T) {
	isolateKnowledgeConfig(t)
	if _, err := SaveRunnerProjectPublicationPolicyMirror(PublicationPolicy{
		Project:   "runner://garage",
		Mode:      PublicationPublish,
		RemoteURL: "https://github.com/DaisyCloverSoftware/garage.git",
	}); err != nil {
		t.Fatal(err)
	}
	task := Task{
		ProjectPath: "runner://garage",
		Review: &TaskReviewResult{
			Changed:           true,
			PublicationStatus: ReviewPublicationPublished,
			PullRequestStatus: ReviewPullRequestAvailable,
			PullRequestNumber: 17,
			PullRequestState:  "open",
		},
	}
	if got, want := ReviewPullRequestURL(task), "https://github.com/DaisyCloverSoftware/garage/pull/17"; got != want {
		t.Fatalf("review URL=%q want %q", got, want)
	}
}
