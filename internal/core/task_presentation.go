package core

import (
	"fmt"
	"regexp"
	"strings"
)

// TaskPresentation is a UI-facing interpretation of durable task state. It is
// intentionally derived from Task rather than persisted separately so desktop,
// MCP and future companion surfaces cannot drift on what the user should do
// next. Publication targets/URLs remain private policy data.
type TaskPresentation struct {
	StatusLabel       string                  `json:"status_label"`
	NextAction        string                  `json:"next_action"`
	ProviderLabel     string                  `json:"provider_label"`
	ReviewBranch      string                  `json:"review_branch,omitempty"`
	ReviewCommit      string                  `json:"review_commit,omitempty"`
	ReviewFiles       int                     `json:"review_files,omitempty"`
	PublicationStatus ReviewPublicationStatus `json:"publication_status,omitempty"`
	Published         bool                    `json:"published,omitempty"`
	PullRequestStatus ReviewPullRequestStatus `json:"pull_request_status,omitempty"`
	PullRequestNumber int                     `json:"pull_request_number,omitempty"`
	PullRequestState  string                  `json:"pull_request_state,omitempty"`
	NeedsHuman        bool                    `json:"needs_human"`
	Terminal          bool                    `json:"terminal"`
}

type TaskDashboardSummary struct {
	Active     int `json:"active"`
	NeedsHuman int `json:"needs_human"`
	Completed  int `json:"completed"`
	Failed     int `json:"failed"`
}

// preparedReviewLine is retained only for v0.7-era persisted task history.
// New executions persist Task.Review and must never depend on parsing worker
// prose to decide whether a review artifact exists.
var preparedReviewLine = regexp.MustCompile(`(?m)Workbench (prepared|published) review branch ([^\s]+) at ([0-9a-fA-F]{7,64})\.`)

func PresentTask(t Task) TaskPresentation {
	provider := strings.TrimSpace(t.ProviderID)
	if provider == "" {
		provider = "Router"
	}
	p := TaskPresentation{ProviderLabel: provider}
	if review := t.Review; review != nil && review.Changed {
		p.ReviewBranch = review.Branch
		p.ReviewCommit = review.Commit
		p.ReviewFiles = len(review.Files)
		p.PublicationStatus = review.PublicationStatus
		p.Published = review.Published || review.PublicationStatus == ReviewPublicationPublished
		p.PullRequestStatus = review.PullRequestStatus
		p.PullRequestNumber = review.PullRequestNumber
		p.PullRequestState = review.PullRequestState
	} else if match := preparedReviewLine.FindStringSubmatch(t.Output); len(match) == 4 {
		// Backward-compatible display for task records created before structured
		// review results existed.
		p.Published = match[1] == "published"
		p.ReviewBranch = match[2]
		p.ReviewCommit = match[3]
		if p.Published {
			p.PublicationStatus = ReviewPublicationPublished
		} else {
			p.PublicationStatus = ReviewPublicationPrepared
		}
	}

	switch t.Status {
	case TaskQueued:
		p.StatusLabel = "Queued"
		p.NextAction = "Workbench will pick this up automatically. You do not need to supervise it."
	case TaskRouting:
		p.StatusLabel = "Choosing worker"
		p.NextAction = "Workbench is choosing the cheapest eligible worker and will continue automatically."
	case TaskRunning:
		p.StatusLabel = "Working"
		p.NextAction = "Leave it running. Come back when it finishes or Workbench genuinely needs you."
	case TaskNeedsAttention:
		p.StatusLabel = "Needs you"
		p.NeedsHuman = true
		p.NextAction = "Answer the question below. Workbench will resume the same task after your decision."
	case TaskCompleted:
		p.StatusLabel = "Ready"
		p.Terminal = true
		switch {
		case p.PullRequestStatus == ReviewPullRequestAvailable && p.PullRequestNumber > 0:
			state := strings.TrimSpace(p.PullRequestState)
			if state == "" {
				state = "ready"
			}
			p.NextAction = fmt.Sprintf("Review GitHub PR #%d (%s). Workbench has finished the coding and delivery work.", p.PullRequestNumber, state)
		case p.ReviewBranch != "" && p.PublicationStatus == ReviewPublicationFailed:
			p.NextAction = fmt.Sprintf("Code is ready for review on %s at %s. Automatic publication did not complete; the prepared review is preserved and does not need to be recoded. Retry review delivery; coding will not run again.", p.ReviewBranch, shortCommit(p.ReviewCommit))
		case p.ReviewBranch != "" && p.Published && p.PullRequestStatus == ReviewPullRequestUnavailable:
			p.NextAction = fmt.Sprintf("The review branch %s is published at %s, but GitHub PR delivery is unavailable. Retry review delivery; coding will not run again.", p.ReviewBranch, shortCommit(p.ReviewCommit))
		case p.ReviewBranch != "" && p.Published:
			p.NextAction = fmt.Sprintf("Ready for review. Workbench published %s at %s.", p.ReviewBranch, shortCommit(p.ReviewCommit))
		case p.ReviewBranch != "":
			p.NextAction = fmt.Sprintf("Ready for review on local branch %s at %s.", p.ReviewBranch, shortCommit(p.ReviewCommit))
		default:
			p.NextAction = "Completed. Read the result below; no further action is required unless you want another change."
		}
	case TaskFailed:
		p.StatusLabel = "Could not finish"
		p.Terminal = true
		p.NextAction = "Read the failure summary below. Workbench already tried the eligible fallback workers before stopping."
	case TaskCancelled:
		p.StatusLabel = "Cancelled"
		p.Terminal = true
		p.NextAction = "This task is stopped. Start a new task when you want to continue."
	default:
		p.StatusLabel = strings.Title(strings.ReplaceAll(string(t.Status), "_", " "))
		p.NextAction = "Review the task details below."
	}
	return p
}

func SummarizeTasks(tasks []Task) TaskDashboardSummary {
	var s TaskDashboardSummary
	for _, t := range tasks {
		switch t.Status {
		case TaskQueued, TaskRouting, TaskRunning:
			s.Active++
		case TaskNeedsAttention:
			s.NeedsHuman++
		case TaskCompleted:
			s.Completed++
		case TaskFailed:
			s.Failed++
		}
	}
	return s
}

func shortCommit(commit string) string {
	commit = strings.TrimSpace(commit)
	if len(commit) > 10 {
		return commit[:10]
	}
	return commit
}
