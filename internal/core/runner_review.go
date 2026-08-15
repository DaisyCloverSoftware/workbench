package core

import (
	"context"
	"errors"
	"strings"
)

type RunnerReviewRequest struct {
	Action  string           `json:"action"`
	Project string           `json:"project"`
	Review  TaskReviewResult `json:"review"`
}

type RunnerReviewResponse struct {
	OK     bool              `json:"ok"`
	Review *TaskReviewResult `json:"review,omitempty"`
	Error  string            `json:"error,omitempty"`
}

// ApplyRunnerReviewRequest is an operator/control-plane operation. It retries
// delivery from an already-verified Workbench review commit on the runner host;
// it never invokes a coding worker and accepts no publication target.
func ApplyRunnerReviewRequest(ctx context.Context, req RunnerReviewRequest) (RunnerReviewResponse, error) {
	if strings.ToLower(strings.TrimSpace(req.Action)) != "retry" {
		return RunnerReviewResponse{}, errors.New("runner review action must be retry")
	}
	if strings.TrimSpace(req.Project) == "" {
		return RunnerReviewResponse{}, errors.New("runner review project is required")
	}
	resolved, err := ResolveRunnerProject(req.Project)
	if err != nil {
		return RunnerReviewResponse{}, err
	}
	updated, retryErr := RetryReviewDelivery(ctx, resolved, req.Review)
	response := RunnerReviewResponse{OK: true, Review: cloneTaskReviewResult(&updated)}
	if retryErr != nil {
		// The structured review state is still useful to the desktop. Keep the
		// wire error generic; raw remote/auth output never crosses this control
		// channel or enters task state.
		response.Error = "review delivery is still unavailable"
	}
	return response, nil
}
