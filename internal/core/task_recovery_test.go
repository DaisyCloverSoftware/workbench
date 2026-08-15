package core

import (
	"testing"
	"time"
)

func TestRecoverInterruptedTasksOnlyQueuesNonTerminalExecutionStates(t *testing.T) {
	future := time.Now().UTC().Add(time.Hour)
	st := DefaultState()
	st.Tasks = []Task{
		{ID: "queued", Status: TaskQueued, ProviderID: "worker"},
		{ID: "routing", Status: TaskRouting, ProviderID: "worker"},
		{ID: "running", Status: TaskRunning, ProviderID: "worker", ConsumesWork: true},
		{ID: "waiting", Status: TaskWaitingRetry, RetryAt: &future, AutoRetryCount: 1},
		{ID: "attention", Status: TaskNeedsAttention, AttentionQuestion: "choose"},
		{ID: "done", Status: TaskCompleted},
		{ID: "failed", Status: TaskFailed},
		{ID: "cancelled", Status: TaskCancelled},
	}
	ids := recoverInterruptedTasks(&st)
	if len(ids) != 3 || ids[0] != "queued" || ids[1] != "routing" || ids[2] != "running" {
		t.Fatalf("unexpected recovered ids: %#v", ids)
	}
	for i := 0; i < 3; i++ {
		if st.Tasks[i].Status != TaskQueued || st.Tasks[i].ProviderID != "" || st.Tasks[i].ConsumesWork || st.Tasks[i].FinishedAt != nil || st.Tasks[i].RetryAt != nil {
			t.Fatalf("task was not reset for routing: %#v", st.Tasks[i])
		}
	}
	if len(st.Tasks[1].Attempts) != 1 || len(st.Tasks[2].Attempts) != 1 {
		t.Fatal("interrupted routing/running tasks should record a recovery attempt")
	}
	if st.Tasks[3].Status != TaskWaitingRetry || st.Tasks[3].RetryAt == nil || !st.Tasks[3].RetryAt.Equal(future) || st.Tasks[3].AutoRetryCount != 1 {
		t.Fatalf("future waiting-retry task changed during ordinary interrupted recovery: %#v", st.Tasks[3])
	}
	if st.Tasks[4].Status != TaskNeedsAttention || st.Tasks[4].AttentionQuestion != "choose" {
		t.Fatal("attention task changed during recovery")
	}
	for _, i := range []int{5, 6, 7} {
		if st.Tasks[i].Status == TaskQueued {
			t.Fatalf("terminal task was reopened: %#v", st.Tasks[i])
		}
	}
}

func TestRecoverWaitingRetryTasksPreservesFutureAndQueuesOverdue(t *testing.T) {
	now := time.Date(2026, 8, 15, 17, 30, 0, 0, time.UTC)
	future := now.Add(5 * time.Minute)
	overdue := now.Add(-time.Minute)
	terminalRetry := now.Add(-time.Hour)
	st := DefaultState()
	st.Tasks = []Task{
		{ID: "future", Status: TaskWaitingRetry, RetryAt: &future, AutoRetryCount: 1},
		{ID: "overdue", Status: TaskWaitingRetry, RetryAt: &overdue, AutoRetryCount: 2, ProviderID: "claude", ConsumesWork: true},
		{ID: "missing", Status: TaskWaitingRetry, AutoRetryCount: 3},
		{ID: "done", Status: TaskCompleted, RetryAt: &terminalRetry},
	}

	retryNow, retryLater := recoverWaitingRetryTasks(&st, now)
	if len(retryNow) != 2 || retryNow[0] != "overdue" || retryNow[1] != "missing" {
		t.Fatalf("unexpected immediate retries: %#v", retryNow)
	}
	if len(retryLater) != 1 || retryLater[0].TaskID != "future" || !retryLater[0].RetryAt.Equal(future) {
		t.Fatalf("unexpected future retry schedule: %#v", retryLater)
	}
	if st.Tasks[0].Status != TaskWaitingRetry || st.Tasks[0].RetryAt == nil || !st.Tasks[0].RetryAt.Equal(future) {
		t.Fatalf("future retry deadline was changed: %#v", st.Tasks[0])
	}
	for _, i := range []int{1, 2} {
		if st.Tasks[i].Status != TaskQueued || st.Tasks[i].RetryAt != nil || st.Tasks[i].ProviderID != "" || st.Tasks[i].ConsumesWork || st.Tasks[i].FinishedAt != nil {
			t.Fatalf("overdue waiting retry was not safely requeued: %#v", st.Tasks[i])
		}
		if len(st.Tasks[i].Attempts) == 0 {
			t.Fatalf("overdue retry recovery was not recorded: %#v", st.Tasks[i])
		}
	}
	if st.Tasks[3].Status != TaskCompleted || st.Tasks[3].RetryAt == nil || !st.Tasks[3].RetryAt.Equal(terminalRetry) {
		t.Fatalf("terminal task was changed by retry recovery: %#v", st.Tasks[3])
	}
}
