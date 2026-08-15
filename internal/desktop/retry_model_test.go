package desktop

import (
	"strings"
	"testing"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestRetryingTaskItemCarriesDeadlineAndLocalNextAttempt(t *testing.T) {
	retryAt := time.Date(2026, 8, 15, 18, 45, 0, 0, time.UTC)
	item := taskItems([]core.Task{{
		ID:       "retrying",
		Title:    "Retry me",
		Status:   core.TaskWaitingRetry,
		RetryAt:  &retryAt,
	}})[0]
	if item.Status != core.TaskWaitingRetry || item.StatusLabel != "Waiting to retry" || item.NeedsHuman || item.Terminal {
		t.Fatalf("retrying task item flags are wrong: %#v", item)
	}
	if item.RetryAt == nil || !item.RetryAt.Equal(retryAt) {
		t.Fatalf("retry deadline was lost: %#v", item)
	}
	if !strings.Contains(item.NextAction, "Next attempt:") {
		t.Fatalf("desktop next action does not show retry deadline: %q", item.NextAction)
	}
}

func TestChooseSelectedTaskTreatsWaitingRetryAsActive(t *testing.T) {
	tasks := []TaskItem{
		{ID: "done", Status: core.TaskCompleted, Terminal: true},
		{ID: "retrying", Status: core.TaskWaitingRetry},
	}
	if got := chooseSelectedTask(tasks, ""); got != "retrying" {
		t.Fatalf("selected task=%q want retrying", got)
	}
}
