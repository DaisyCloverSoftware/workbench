package mcp

import (
	"context"
	"errors"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

const (
	defaultOperationWait = 120 * time.Second
	maxOperationWait     = 10 * time.Minute
	operationWaitPoll    = 250 * time.Millisecond
)

type operationWaitResult struct {
	Task         core.Task `json:"task"`
	WaitTimedOut bool      `json:"wait_timed_out,omitempty"`
}

func operationWaitDuration(seconds int) time.Duration {
	if seconds <= 0 {
		return defaultOperationWait
	}
	if seconds > int(maxOperationWait/time.Second) {
		seconds = int(maxOperationWait / time.Second)
	}
	return time.Duration(seconds) * time.Second
}

func operationTerminal(status core.TaskStatus) bool {
	switch status {
	case core.TaskCompleted, core.TaskFailed, core.TaskCancelled, core.TaskNeedsAttention:
		return true
	default:
		return false
	}
}

// awaitOperation turns the durable operations task into an app-friendly wait:
// ChatGPT can hand Workbench a machine-side objective, then block for a bounded
// period instead of repeatedly polling or asking the human to watch progress.
// The underlying task remains durable if the tool call times out or is cancelled.
func awaitOperation(ctx context.Context, eng *core.Engine, taskID string, timeout time.Duration) (operationWaitResult, error) {
	if eng == nil {
		return operationWaitResult{}, errors.New("Workbench engine is unavailable")
	}
	if timeout <= 0 || timeout > maxOperationWait {
		timeout = maxOperationWait
	}

	read := func() (core.Task, error) {
		task, ok := eng.Task(taskID)
		if !ok {
			return core.Task{}, errors.New("operation task not found")
		}
		if !core.IsOperationsTask(task) {
			return core.Task{}, errors.New("await_operation accepts machine-side operations only")
		}
		return task, nil
	}

	task, err := read()
	if err != nil {
		return operationWaitResult{}, err
	}
	if operationTerminal(task.Status) {
		return operationWaitResult{Task: task}, nil
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	ticker := time.NewTicker(operationWaitPoll)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return operationWaitResult{}, ctx.Err()
		case <-timer.C:
			task, err := read()
			if err != nil {
				return operationWaitResult{}, err
			}
			return operationWaitResult{Task: task, WaitTimedOut: true}, nil
		case <-ticker.C:
			task, err := read()
			if err != nil {
				return operationWaitResult{}, err
			}
			if operationTerminal(task.Status) {
				return operationWaitResult{Task: task}, nil
			}
		}
	}
}
