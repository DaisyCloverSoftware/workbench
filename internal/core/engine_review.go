package core

import (
	"context"
	"errors"
	"strings"
	"time"
)

// RetryTaskReviewDelivery retries only Workbench control-plane delivery for a
// completed task. It never routes to a provider or reconstructs a worker prompt.
func (e *Engine) RetryTaskReviewDelivery(taskID string) error {
	task, ok := e.Task(taskID)
	if !ok {
		return errors.New("task not found")
	}
	if task.Status != TaskCompleted || task.Review == nil || !task.Review.Changed {
		return errors.New("task has no completed review to deliver")
	}
	prefs := e.State().Preferences
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	updated := *cloneTaskReviewResult(task.Review)
	var retryErr error
	if task.ProviderID == "workbench-runner" {
		host := strings.TrimSpace(prefs.OpenClawSSHHost)
		if host == "" {
			return errors.New("configured Workbench runner is unavailable")
		}
		response, err := RunRunnerReviewSSH(ctx, host, RunnerReviewRequest{
			Action:  "retry",
			Project: task.ProjectPath,
			Review:  updated,
		})
		if response.Review != nil {
			updated = *cloneTaskReviewResult(response.Review)
		}
		if err != nil {
			retryErr = errors.New("runner review delivery is unavailable")
		} else if strings.TrimSpace(response.Error) != "" {
			retryErr = errors.New("review delivery is still unavailable")
		}
	} else {
		var err error
		updated, err = RetryReviewDelivery(ctx, task.ProjectPath, updated)
		if err != nil {
			retryErr = errors.New("review delivery is still unavailable")
		}
	}

	e.mu.Lock()
	i := e.taskIndexLocked(taskID)
	if i < 0 {
		e.mu.Unlock()
		return errors.New("task not found")
	}
	if e.state.Tasks[i].Status != TaskCompleted {
		e.mu.Unlock()
		return errors.New("task is no longer completed")
	}
	e.state.Tasks[i].Review = cloneTaskReviewResult(&updated)
	e.state.Tasks[i].UpdatedAt = time.Now()
	outcome := "review delivery retry completed"
	if updated.PublicationStatus == ReviewPublicationFailed {
		outcome = "review delivery retry: publication still unavailable"
	} else if updated.PullRequestStatus == ReviewPullRequestUnavailable {
		outcome = "review delivery retry: branch published; GitHub PR unavailable"
	} else if updated.PullRequestStatus == ReviewPullRequestAvailable {
		outcome = "review delivery retry: GitHub PR available"
	}
	e.state.Tasks[i].Attempts = append(e.state.Tasks[i].Attempts, outcome)
	st := cloneState(e.state)
	e.mu.Unlock()
	if err := e.store.Save(st); err != nil {
		return err
	}
	e.notify()
	if retryErr != nil {
		return retryErr
	}
	if updated.PublicationStatus == ReviewPublicationPublished && updated.PullRequestStatus == ReviewPullRequestUnavailable {
		return errors.New("review branch is published but GitHub PR delivery is still unavailable")
	}
	return nil
}
