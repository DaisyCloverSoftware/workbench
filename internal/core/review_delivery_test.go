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

func TestGitHubRepoSlugAcceptsOnlyCanonicalGitHubRemotes(t *testing.T) {
	tests := []struct {
		remote string
		want   string
		ok     bool
	}{
		{"https://github.com/DaisyCloverSoftware/workbench.git", "DaisyCloverSoftware/workbench", true},
		{"ssh://git@github.com/DaisyCloverSoftware/workbench.git", "DaisyCloverSoftware/workbench", true},
		{"git@github.com:DaisyCloverSoftware/workbench.git", "DaisyCloverSoftware/workbench", true},
		{"https://user@github.com/DaisyCloverSoftware/workbench.git", "", false},
		{"https://github.example.com/DaisyCloverSoftware/workbench.git", "", false},
		{"git@github.example.com:DaisyCloverSoftware/workbench.git", "", false},
		{"https://github.com/DaisyCloverSoftware/workbench/extra", "", false},
		{"/local/review.git", "", false},
	}
	for _, tc := range tests {
		got, ok := githubRepoSlug(tc.remote)
		if got != tc.want || ok != tc.ok {
			t.Fatalf("githubRepoSlug(%q)=(%q,%t), want (%q,%t)", tc.remote, got, ok, tc.want, tc.ok)
		}
	}
}

func TestDeliverGitHubPullRequestReusesExactExistingPR(t *testing.T) {
	review := TaskReviewResult{
		Changed:           true,
		BaseBranch:        "main",
		Branch:            "workbench/task-123",
		Commit:            "abcdef0123456789",
		PublicationStatus: ReviewPublicationPublished,
		Published:         true,
	}
	run := func(ctx context.Context, args ...string) ([]byte, error) {
		if len(args) >= 2 && args[0] == "pr" && args[1] == "list" {
			return json.Marshal([]githubPullRequest{{
				Number:      42,
				URL:         "https://github.com/DaisyCloverSoftware/workbench/pull/42",
				State:       "OPEN",
				HeadRefName: review.Branch,
				HeadRefOID:  review.Commit,
				BaseRefName: "main",
			}})
		}
		t.Fatalf("unexpected gh command: %v", args)
		return nil, nil
	}
	got := deliverGitHubPullRequestWithRunner(context.Background(), "DaisyCloverSoftware/workbench", review, run)
	if got.PullRequestStatus != ReviewPullRequestAvailable || got.PullRequestNumber != 42 || got.PullRequestState != "open" {
		t.Fatalf("unexpected PR delivery result: %#v", got)
	}
}

func TestDeliverGitHubPullRequestCreatesThenVerifiesPR(t *testing.T) {
	review := TaskReviewResult{
		Changed:           true,
		BaseBranch:        "develop",
		Branch:            "workbench/task-create",
		Commit:            "0123456789abcdef",
		PublicationStatus: ReviewPublicationPublished,
		Published:         true,
	}
	lists := 0
	creates := 0
	run := func(ctx context.Context, args ...string) ([]byte, error) {
		if len(args) >= 2 && args[0] == "pr" && args[1] == "list" {
			lists++
			if lists == 1 {
				return []byte("[]"), nil
			}
			return json.Marshal([]githubPullRequest{{
				Number:      77,
				URL:         "https://github.com/DaisyCloverSoftware/workbench/pull/77",
				State:       "OPEN",
				HeadRefName: review.Branch,
				HeadRefOID:  review.Commit,
				BaseRefName: "develop",
			}})
		}
		if len(args) >= 2 && args[0] == "pr" && args[1] == "create" {
			creates++
			joined := strings.Join(args, " ")
			if !strings.Contains(joined, "--base develop") || !strings.Contains(joined, "--head "+review.Branch) {
				t.Fatalf("PR create did not use recorded review refs: %v", args)
			}
			if strings.Contains(strings.ToLower(joined), "publication target") {
				t.Fatalf("PR body leaked control-plane target language: %v", args)
			}
			return []byte("https://github.com/DaisyCloverSoftware/workbench/pull/77"), nil
		}
		t.Fatalf("unexpected gh command: %v", args)
		return nil, nil
	}
	got := deliverGitHubPullRequestWithRunner(context.Background(), "DaisyCloverSoftware/workbench", review, run)
	if creates != 1 || lists != 2 {
		t.Fatalf("gh calls: creates=%d lists=%d", creates, lists)
	}
	if got.PullRequestStatus != ReviewPullRequestAvailable || got.PullRequestNumber != 77 {
		t.Fatalf("unexpected PR delivery result: %#v", got)
	}
}

func TestDeliverGitHubPullRequestRefusesStaleBranchPR(t *testing.T) {
	review := TaskReviewResult{
		Changed:           true,
		BaseBranch:        "main",
		Branch:            "workbench/task-stale",
		Commit:            "aaaaaaaaaaaaaaaa",
		PublicationStatus: ReviewPublicationPublished,
		Published:         true,
	}
	creates := 0
	run := func(ctx context.Context, args ...string) ([]byte, error) {
		if args[0] == "pr" && args[1] == "list" {
			return json.Marshal([]githubPullRequest{{
				Number:      9,
				URL:         "https://github.com/DaisyCloverSoftware/workbench/pull/9",
				State:       "OPEN",
				HeadRefName: review.Branch,
				HeadRefOID:  "bbbbbbbbbbbbbbbb",
				BaseRefName: "main",
			}})
		}
		if args[0] == "pr" && args[1] == "create" {
			creates++
		}
		return nil, nil
	}
	got := deliverGitHubPullRequestWithRunner(context.Background(), "DaisyCloverSoftware/workbench", review, run)
	if creates != 0 {
		t.Fatal("stale branch PR caused a duplicate PR create attempt")
	}
	if got.PullRequestStatus != ReviewPullRequestUnavailable || got.PullRequestNumber != 0 {
		t.Fatalf("stale PR was incorrectly attached: %#v", got)
	}
}

func TestRetryReviewDeliveryPublishesExistingPreparedReviewWithoutAI(t *testing.T) {
	isolateKnowledgeConfig(t)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	repo := initPrepareTestRepo(t)
	ctx := context.Background()

	if _, err := SavePublicationPolicy(PublicationPolicy{Project: repo, Mode: PublicationPrepare}); err != nil {
		t.Fatal(err)
	}
	ws, err := CreateTaskWorkspace(ctx, repo, "task-retry-delivery")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws.Workspace, "tracked.txt"), []byte("ready for retry\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	review, err := FinalizeTaskWorkspace(ctx, ws)
	if err != nil {
		t.Fatal(err)
	}
	if review.PublicationStatus != ReviewPublicationPrepared {
		t.Fatalf("initial review status=%q", review.PublicationStatus)
	}

	remote := filepath.Join(t.TempDir(), "review.git")
	cmd := exec.Command("git", "init", "--bare", "-q", remote)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %v\n%s", err, out)
	}
	if _, err := SavePublicationPolicy(PublicationPolicy{Project: repo, Mode: PublicationPublish, RemoteURL: remote}); err != nil {
		t.Fatal(err)
	}
	got, err := RetryReviewDelivery(ctx, repo, review)
	if err != nil {
		t.Fatal(err)
	}
	if got.PublicationStatus != ReviewPublicationPublished || !got.Published || got.PullRequestStatus != ReviewPullRequestNotApplicable {
		t.Fatalf("unexpected retry result: %#v", got)
	}
	out, err := exec.Command("git", "--git-dir", remote, "rev-parse", "refs/heads/"+review.Branch).CombinedOutput()
	if err != nil {
		t.Fatalf("read retried remote branch: %v\n%s", err, out)
	}
	if strings.TrimSpace(string(out)) != review.Commit {
		t.Fatalf("retried branch=%q commit=%q", strings.TrimSpace(string(out)), review.Commit)
	}
}
