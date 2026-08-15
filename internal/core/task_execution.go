package core

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type ReviewPublicationStatus string

const (
	ReviewPublicationPrepared ReviewPublicationStatus = "prepared"
	ReviewPublicationPublished ReviewPublicationStatus = "published"
	ReviewPublicationFailed ReviewPublicationStatus = "publication_failed"
)

type TaskReviewResult struct {
	Changed           bool                    `json:"changed"`
	BaseRevision      string                  `json:"base_revision,omitempty"`
	Fingerprint       string                  `json:"fingerprint,omitempty"`
	Branch            string                  `json:"branch,omitempty"`
	Commit            string                  `json:"commit,omitempty"`
	Files             []string                `json:"files,omitempty"`
	PublicationStatus ReviewPublicationStatus `json:"publication_status,omitempty"`
	Published         bool                    `json:"published,omitempty"`
	AlreadyPresent    bool                    `json:"already_present,omitempty"`
}

// RunProviderIsolated gives a local coding worker a durable Workbench-owned
// task worktree rather than the user's source checkout. A workspace survives
// ordinary worker failure and human-attention pauses so another eligible worker
// or the resumed task can continue the same isolated edits.
func RunProviderIsolated(ctx context.Context, p Provider, task Task, prefs Preferences) (RunResult, error) {
	// The cluster runner is itself a Workbench control plane. Isolation must be
	// created on that host after its project path has been resolved, not on the
	// desktop that is sending the request.
	if p.ID == "workbench-runner" {
		return RunProvider(ctx, p, task, prefs)
	}
	// Direct SSH OpenClaw cannot safely consume a desktop-local worktree path.
	// When a runner host is configured, require the structured Workbench runner
	// route instead of falling back to an unisolated remote coding session.
	if p.ID == "openclaw" && strings.TrimSpace(prefs.OpenClawSSHHost) != "" {
		return RunResult{Retryable: true}, errors.New("remote OpenClaw coding requires the Workbench cluster runner so task isolation is enforced on the execution host")
	}

	ws, err := CreateTaskWorkspace(ctx, task.ProjectPath, task.ID)
	if err != nil {
		return RunResult{}, fmt.Errorf("create isolated task workspace: %w", err)
	}
	workerTask := task
	workerTask.memoryProjectPath = taskMemoryProject(task)
	workerTask.ProjectPath = ws.Workspace
	res, runErr := RunProvider(ctx, p, workerTask, prefs)
	if strings.TrimSpace(res.Attention) != "" || runErr != nil {
		return res, runErr
	}

	review, err := FinalizeTaskWorkspace(ctx, ws)
	if err != nil {
		return res, fmt.Errorf("finalize isolated task workspace: %w", err)
	}
	if review.Changed {
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
		if strings.TrimSpace(res.Output) == "" {
			res.Output = status
		} else {
			res.Output = strings.TrimSpace(res.Output) + "\n\n" + status
		}
	}
	return res, nil
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
