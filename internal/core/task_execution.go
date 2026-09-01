package core

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type ReviewPublicationStatus string
type ReviewPullRequestStatus string

const (
	ReviewPublicationPrepared ReviewPublicationStatus = "prepared"
	ReviewPublicationPublished ReviewPublicationStatus = "published"
	ReviewPublicationFailed   ReviewPublicationStatus = "publication_failed"

	ReviewPullRequestNotApplicable ReviewPullRequestStatus = "not_applicable"
	ReviewPullRequestAvailable     ReviewPullRequestStatus = "available"
	ReviewPullRequestUnavailable   ReviewPullRequestStatus = "unavailable"
)

type TaskReviewResult struct {
	Changed           bool                    `json:"changed"`
	BaseRevision      string                  `json:"base_revision,omitempty"`
	BaseBranch        string                  `json:"base_branch,omitempty"`
	Fingerprint       string                  `json:"fingerprint,omitempty"`
	Branch            string                  `json:"branch,omitempty"`
	Commit            string                  `json:"commit,omitempty"`
	Files             []string                `json:"files,omitempty"`
	PublicationStatus ReviewPublicationStatus `json:"publication_status,omitempty"`
	Published         bool                    `json:"published,omitempty"`
	AlreadyPresent    bool                    `json:"already_present,omitempty"`
	PullRequestStatus ReviewPullRequestStatus `json:"pull_request_status,omitempty"`
	PullRequestNumber int                     `json:"pull_request_number,omitempty"`
	PullRequestState  string                  `json:"pull_request_state,omitempty"`
}

// RunProviderIsolated gives a local coding worker a durable Workbench-owned
// task worktree rather than the user's source checkout. A workspace survives
// ordinary worker failure and human-attention pauses so another eligible worker
// or the resumed task can continue the same isolated edits.
//
// ChatGPT-originated development work has a stronger contract: ChatGPT owns
// source changes, Git/GitHub operations, PRs, CI and GitHub Actions. A private
// relay continuation is the explicit development-continuity exception and must
// carry a valid HMAC seal before any autonomous coding worker may run.
//
// OpenClaw is a separate owner-selected machine-operations mode. An Operations
// task may reach either the local OpenClaw adapter or the cluster runner only
// when durable task state proves explicit owner authorization. Provider
// availability, task mode, or the operations marker alone is not authorization.
func RunProviderIsolated(ctx context.Context, p Provider, task Task, prefs Preferences) (RunResult, error) {
	if IsOperationsTask(task) && !task.OpenClawOwnerAuthorized {
		return RunResult{}, errors.New("OpenClaw authorization denied: Operations task lacks durable explicit owner authorization naming OpenClaw")
	}
	if p.ID == "openclaw" && (!IsOperationsTask(task) || !task.OpenClawOwnerAuthorized) {
		return RunResult{}, errors.New("OpenClaw is owner-opt-in only and cannot be used as an automatic development or fallback provider")
	}
	if strings.EqualFold(strings.TrimSpace(task.Origin), "chatgpt-mcp") && !IsOperationsTask(task) {
		continuation, ok := ValidatePrivateRelayContinuationIntent(task.Intent, task.ProjectPath, prefs.MCPToken)
		if !ok {
			return RunResult{}, errors.New("ChatGPT owns coding, Git/GitHub, pull requests, CI and GitHub Actions; Workbench will not delegate that development loop to an autonomous worker")
		}
		// Keep proof material only in durable control-plane state. Workers receive
		// the exact continuation objective and never see relay authentication data.
		task.Intent = continuation
	}
	if IsOperationsTask(task) && p.ID != "openclaw" && p.ID != "workbench-runner" {
		return RunResult{}, fmt.Errorf("provider %s is not applicable to an explicitly owner-authorized OpenClaw operations task; operations are reserved for the cluster runner/OpenClaw operator lane", p.Name)
	}

	// A runner:// project deliberately has no desktop-local worktree. It may only
	// be executed by the Workbench cluster runner, which resolves the logical
	// reference inside its authorised runner root and creates isolation there.
	// This is an eligibility refusal, not a provider outage, so do not mark a
	// perfectly healthy local provider retryable/cooling merely because the
	// project lives on another host.
	if IsRunnerProjectReference(task.ProjectPath) && p.ID != "workbench-runner" {
		return RunResult{}, errors.New("cluster project requires the configured Workbench runner")
	}
	// The cluster runner is itself a Workbench control plane. Isolation must be
	// created on that host after its project path has been resolved, not on the
	// desktop that is sending the request.
	if p.ID == "workbench-runner" {
		return RunProvider(ctx, p, task, prefs)
	}
	// Direct SSH OpenClaw cannot safely consume a desktop-local worktree path.
	// When a runner host is configured, require the structured Workbench runner
	// route instead of using an unisolated remote session.
	if p.ID == "openclaw" && strings.TrimSpace(prefs.OpenClawSSHHost) != "" {
		return RunResult{Retryable: true}, errors.New("remote OpenClaw execution requires the Workbench cluster runner so task isolation is enforced on the execution host")
	}

	var ws TaskWorkspace
	var err error
	if IsOperationsTask(task) {
		ws, err = CreateOperationsTaskWorkspace(ctx, task.ProjectPath, task.ID)
	} else {
		ws, err = CreateTaskWorkspace(ctx, task.ProjectPath, task.ID)
	}
	if err != nil {
		return RunResult{}, fmt.Errorf("create isolated task workspace: %w", err)
	}
	workerTask := task
	workerTask.memoryProjectPath = taskMemoryProject(task)
	workerTask.ProjectPath = ws.Workspace

	if IsOperationsTask(task) {
		res, runErr := RunOpenClawOperationSupervised(ctx, p, workerTask, prefs)
		if strings.TrimSpace(res.Attention) != "" || runErr != nil {
			return boundRunResultForPersistence(res), runErr
		}
		inspection, inspectErr := InspectChangeset(ctx, ws.Workspace)
		if inspectErr != nil {
			return boundRunResultForPersistence(res), fmt.Errorf("verify OpenClaw operations workspace remained source-clean: %w", inspectErr)
		}
		if !inspection.Clean {
			removeErr := RemoveTaskWorkspace(ctx, ws.Project, ws.TaskID)
			message := "OpenClaw attempted repository/source changes during an operations task. Workbench isolated and discarded those edits because ChatGPT is the coder; the operational result was not accepted as complete."
			if removeErr != nil {
				message += " The isolated workspace also could not be removed automatically."
			}
			if strings.TrimSpace(res.Output) != "" {
				res.Output = strings.TrimSpace(res.Output) + "\n\n" + message
			} else {
				res.Output = message
			}
			return boundRunResultForPersistence(res), errors.New("OpenClaw operations lane crossed the source-code boundary")
		}
		if err := RemoveTaskWorkspace(ctx, ws.Project, ws.TaskID); err != nil {
			return boundRunResultForPersistence(res), fmt.Errorf("remove clean operations workspace: %w", err)
		}
		_ = DeleteTaskProviderSessions(task.ID)
		return boundRunResultForPersistence(res), nil
	}

	res, runErr := RunProvider(ctx, p, workerTask, prefs)
	if strings.TrimSpace(res.Attention) != "" || runErr != nil {
		return boundRunResultForPersistence(res), runErr
	}

	review, err := FinalizeTaskWorkspace(ctx, ws)
	if err != nil {
		return boundRunResultForPersistence(res), fmt.Errorf("finalize isolated task workspace: %w", err)
	}
	if review.Changed {
		if review.PublicationStatus == ReviewPublicationPublished {
			review = DeliverGitHubPullRequest(ctx, task.ProjectPath, review)
		}
		res.Review = cloneTaskReviewResult(&review)
		status := fmt.Sprintf("Workbench prepared review branch %s at %s.", review.Branch, review.Commit)
		switch review.PublicationStatus {
		case ReviewPublicationPublished:
			status = fmt.Sprintf("Workbench published review branch %s at %s.", review.Branch, review.Commit)
			if review.AlreadyPresent {
				status += " The same prepared commit was already present on the publication target."
			}
		case ReviewPublicationFailed:
			status += " Automatic review publication did not complete; the verified prepared review remains available locally and coding will not be rerun."
		}
		if review.PullRequestStatus == ReviewPullRequestAvailable && review.PullRequestNumber > 0 {
			status += fmt.Sprintf(" GitHub review PR #%d is available.", review.PullRequestNumber)
		} else if review.PullRequestStatus == ReviewPullRequestUnavailable {
			status += " The review branch is safe, but GitHub PR delivery is currently unavailable and can be retried without recoding."
		}
		if strings.TrimSpace(res.Output) == "" {
			res.Output = status
		} else {
			res.Output = strings.TrimSpace(res.Output) + "\n\n" + status
		}
	}
	// Provider sessions exist only to accelerate an unfinished Workbench task.
	// Once review finalisation succeeds, drop all host-local session pointers for
	// this task; worker transcripts remain provider-managed and never enter Task.
	_ = DeleteTaskProviderSessions(task.ID)
	return boundRunResultForPersistence(res), nil
}

// FinalizeTaskWorkspace turns successful worker edits into a Workbench-owned
// review commit without modifying the user's source checkout. Publication is
// controlled only by the private local policy; when no policy exists, the safe
// default is a local prepared branch. Once a deterministic local review commit
// exists, publication errors are recorded as control-plane state instead of
// failing the coding task and routing the same implementation to another AI.
func FinalizeTaskWorkspace(ctx context.Context, ws TaskWorkspace) (TaskReviewResult, error) {
	if !validTaskWorkspace(ctx, ws) {
		return TaskReviewResult{}, errors.New("task workspace is no longer valid")
	}
	head, err := runGitLimited(ctx, ws.Workspace, 16<<10, "rev-parse", "HEAD")
	if err != nil {
		return TaskReviewResult{}, err
	}
	if strings.TrimSpace(head) != strings.TrimSpace(ws.BaseRevision) {
		return TaskReviewResult{}, errors.New("worker created a commit inside its isolated workspace; Workbench requires workers to leave review commits to the control plane")
	}
	inspection, err := InspectChangeset(ctx, ws.Workspace)
	if err != nil {
		return TaskReviewResult{}, err
	}
	if inspection.Clean {
		if err := RemoveTaskWorkspace(ctx, ws.Project, ws.TaskID); err != nil {
			return TaskReviewResult{}, err
		}
		return TaskReviewResult{}, nil
	}

	prepared, err := PrepareChangeset(ctx, ws.Workspace, ws.TaskID)
	if err != nil {
		return TaskReviewResult{}, err
	}
	result := TaskReviewResult{
		Changed:           true,
		BaseRevision:      prepared.BaseRevision,
		BaseBranch:        ws.BaseBranch,
		Fingerprint:       prepared.Fingerprint,
		Branch:            prepared.Branch,
		Commit:            prepared.Commit,
		Files:             append([]string(nil), prepared.Files...),
		PublicationStatus: ReviewPublicationPrepared,
	}

	policy, configured, policyErr := PublicationPolicyFor(ws.Project)
	if policyErr != nil {
		result.PublicationStatus = ReviewPublicationFailed
	} else if configured && policy.Mode == PublicationPublish {
		published, publishErr := PublishPreparedChangeset(ctx, prepared, policy.RemoteURL)
		if publishErr != nil {
			result.PublicationStatus = ReviewPublicationFailed
		} else {
			result.PublicationStatus = ReviewPublicationPublished
			result.Published = true
			result.AlreadyPresent = published.AlreadyPresent
		}
	}
	if err := RemoveTaskWorkspace(ctx, ws.Project, ws.TaskID); err != nil {
		return TaskReviewResult{}, err
	}
	return result, nil
}

func cloneTaskReviewResult(review *TaskReviewResult) *TaskReviewResult {
	if review == nil {
		return nil
	}
	cloned := *review
	cloned.Files = append([]string(nil), review.Files...)
	return &cloned
}
