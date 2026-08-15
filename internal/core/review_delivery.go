package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var githubRepoPart = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

type ghCommandRunner func(context.Context, ...string) ([]byte, error)

// DeliverGitHubPullRequest optionally attaches a real GitHub pull request to an
// already-published Workbench review branch. It reads the private publication
// target from operator policy and stores only PR number/state/status in the
// task-facing review result. Repository URLs and raw gh/auth errors never enter
// task state.
func DeliverGitHubPullRequest(ctx context.Context, project string, review TaskReviewResult) TaskReviewResult {
	result := *cloneTaskReviewResult(&review)
	if !result.Changed || result.PublicationStatus != ReviewPublicationPublished {
		return result
	}
	policy, configured, err := PublicationPolicyFor(project)
	if err != nil || !configured || policy.Mode != PublicationPublish {
		result.PullRequestStatus = ReviewPullRequestUnavailable
		return result
	}
	slug, ok := githubRepoSlug(policy.RemoteURL)
	if !ok {
		result.PullRequestStatus = ReviewPullRequestNotApplicable
		return result
	}
	if _, err := exec.LookPath("gh"); err != nil {
		result.PullRequestStatus = ReviewPullRequestUnavailable
		return result
	}
	return deliverGitHubPullRequestWithRunner(ctx, slug, result, runGHCommand)
}

func deliverGitHubPullRequestWithRunner(ctx context.Context, slug string, review TaskReviewResult, run ghCommandRunner) TaskReviewResult {
	result := *cloneTaskReviewResult(&review)
	result.PullRequestStatus = ReviewPullRequestUnavailable
	result.PullRequestNumber = 0
	result.PullRequestState = ""
	if !validGitHubRepoSlug(slug) || strings.TrimSpace(result.Branch) == "" || strings.TrimSpace(result.Commit) == "" {
		return result
	}

	base := strings.TrimSpace(result.BaseBranch)
	if base == "" {
		out, err := run(ctx, "repo", "view", slug, "--json", "defaultBranchRef")
		if err != nil {
			return result
		}
		var repo struct {
			DefaultBranchRef struct {
				Name string `json:"name"`
			} `json:"defaultBranchRef"`
		}
		if json.Unmarshal(out, &repo) != nil {
			return result
		}
		base = strings.TrimSpace(repo.DefaultBranchRef.Name)
		if base == "" {
			return result
		}
	}

	existing, found, safe := findPullRequest(ctx, slug, base, result.Branch, result.Commit, run)
	if !safe {
		return result
	}
	if found {
		return attachPullRequest(result, existing)
	}

	body := "Prepared automatically by Workbench for human review.\n\nWorkbench verified this review branch before publishing it. Coding workers cannot push branches or create pull requests directly."
	if _, err := run(ctx,
		"pr", "create",
		"--repo", slug,
		"--base", base,
		"--head", result.Branch,
		"--title", "Workbench prepared review",
		"--body", body,
	); err != nil {
		return result
	}
	created, found, safe := findPullRequest(ctx, slug, base, result.Branch, result.Commit, run)
	if !safe || !found {
		return result
	}
	return attachPullRequest(result, created)
}

type githubPullRequest struct {
	Number      int    `json:"number"`
	URL         string `json:"url"`
	State       string `json:"state"`
	HeadRefName string `json:"headRefName"`
	HeadRefOID  string `json:"headRefOid"`
	BaseRefName string `json:"baseRefName"`
}

func findPullRequest(ctx context.Context, slug, base, branch, commit string, run ghCommandRunner) (githubPullRequest, bool, bool) {
	out, err := run(ctx,
		"pr", "list",
		"--repo", slug,
		"--head", branch,
		"--state", "all",
		"--json", "number,url,state,headRefName,headRefOid,baseRefName",
		"--limit", "20",
	)
	if err != nil {
		return githubPullRequest{}, false, false
	}
	var prs []githubPullRequest
	if json.Unmarshal(out, &prs) != nil {
		return githubPullRequest{}, false, false
	}
	if len(prs) == 0 {
		return githubPullRequest{}, false, true
	}
	for _, pr := range prs {
		if pr.Number <= 0 || strings.TrimSpace(pr.URL) == "" {
			continue
		}
		if strings.TrimSpace(pr.HeadRefName) != branch || strings.TrimSpace(pr.HeadRefOID) != commit || strings.TrimSpace(pr.BaseRefName) != base {
			continue
		}
		if !validPullRequestURL(pr.URL, slug, pr.Number) {
			return githubPullRequest{}, false, false
		}
		return pr, true, true
	}
	// A PR already exists for the Workbench branch but does not point at the
	// exact recorded review commit/base. Refuse to create or link a misleading
	// review artifact.
	return githubPullRequest{}, false, false
}

func attachPullRequest(review TaskReviewResult, pr githubPullRequest) TaskReviewResult {
	review.PullRequestStatus = ReviewPullRequestAvailable
	review.PullRequestNumber = pr.Number
	review.PullRequestState = strings.ToLower(strings.TrimSpace(pr.State))
	return review
}

func runGHCommand(ctx context.Context, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gh", args...)
	cmd.Env = append(os.Environ(), "GH_PROMPT_DISABLED=1", "GIT_TERMINAL_PROMPT=0", "PAGER=cat", "GH_PAGER=cat")
	configureChildProcess(cmd, false)
	stdout := &limitedCapture{limit: 1 << 20}
	stderr := &limitedCapture{limit: 256 << 10}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return nil, errors.New("GitHub review delivery unavailable")
	}
	if stdout.exceeded || stderr.exceeded {
		return nil, errors.New("GitHub review delivery response too large")
	}
	return []byte(strings.TrimSpace(stdout.String())), nil
}

func githubRepoSlug(remote string) (string, bool) {
	remote = strings.TrimSpace(remote)
	if remote == "" {
		return "", false
	}
	if !strings.Contains(remote, "://") && scpStyleGitRemote.MatchString(remote) {
		parts := strings.SplitN(remote, ":", 2)
		hostPart := parts[0]
		if at := strings.LastIndex(hostPart, "@"); at >= 0 {
			hostPart = hostPart[at+1:]
		}
		if !strings.EqualFold(hostPart, "github.com") {
			return "", false
		}
		return normalizeGitHubSlug(parts[1])
	}
	u, err := url.Parse(remote)
	if err != nil || !strings.EqualFold(u.Hostname(), "github.com") || u.RawQuery != "" || u.Fragment != "" {
		return "", false
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		if u.User != nil {
			return "", false
		}
	case "ssh":
		if u.User != nil {
			if _, hasPassword := u.User.Password(); hasPassword {
				return "", false
			}
		}
	default:
		return "", false
	}
	return normalizeGitHubSlug(u.Path)
}

func normalizeGitHubSlug(path string) (string, bool) {
	path = strings.Trim(strings.TrimSpace(path), "/")
	path = strings.TrimSuffix(path, ".git")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || !githubRepoPart.MatchString(parts[0]) || !githubRepoPart.MatchString(parts[1]) || parts[0] == "." || parts[1] == "." {
		return "", false
	}
	return parts[0] + "/" + parts[1], true
}

func validGitHubRepoSlug(slug string) bool {
	_, ok := normalizeGitHubSlug(slug)
	return ok && !strings.HasSuffix(strings.TrimSpace(slug), ".git")
}

func validPullRequestURL(raw, slug string, number int) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || !strings.EqualFold(u.Hostname(), "github.com") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return false
	}
	want := "/" + slug + "/pull/" + strconv.Itoa(number)
	return u.Path == want
}

// ReviewPullRequestURL derives the clickable GitHub review link from private
// local operator policy. The URL is never persisted in Task/RunResult JSON.
func ReviewPullRequestURL(task Task) string {
	if task.Review == nil || task.Review.PullRequestStatus != ReviewPullRequestAvailable || task.Review.PullRequestNumber <= 0 {
		return ""
	}
	policy, configured, err := PublicationPolicyFor(task.ProjectPath)
	if err != nil || !configured || policy.Mode != PublicationPublish {
		return ""
	}
	slug, ok := githubRepoSlug(policy.RemoteURL)
	if !ok {
		return ""
	}
	return fmt.Sprintf("https://github.com/%s/pull/%d", slug, task.Review.PullRequestNumber)
}

// RetryReviewDelivery republishes an already-verified prepared review from its
// structured provenance and then re-attempts GitHub PR delivery. No AI worker
// is invoked and no publication target is accepted from the task/request.
func RetryReviewDelivery(ctx context.Context, project string, review TaskReviewResult) (TaskReviewResult, error) {
	result := *cloneTaskReviewResult(&review)
	if !result.Changed || result.Branch == "" || result.Commit == "" || result.BaseRevision == "" || result.Fingerprint == "" || len(result.Files) == 0 {
		return result, errors.New("task review is incomplete and cannot be retried")
	}
	policy, configured, err := PublicationPolicyFor(project)
	if err != nil || !configured || policy.Mode != PublicationPublish {
		result.PublicationStatus = ReviewPublicationFailed
		result.Published = false
		return result, errors.New("review publication policy is unavailable")
	}
	prepared := PreparedChangeset{
		Project:      project,
		BaseRevision: result.BaseRevision,
		Fingerprint:  result.Fingerprint,
		Branch:       result.Branch,
		Commit:       result.Commit,
		Files:        append([]string(nil), result.Files...),
	}
	published, publishErr := PublishPreparedChangeset(ctx, prepared, policy.RemoteURL)
	if publishErr != nil {
		result.PublicationStatus = ReviewPublicationFailed
		result.Published = false
		return result, errors.New("review publication is still unavailable")
	}
	result.PublicationStatus = ReviewPublicationPublished
	result.Published = true
	result.AlreadyPresent = published.AlreadyPresent
	result = DeliverGitHubPullRequest(ctx, project, result)
	return result, nil
}
