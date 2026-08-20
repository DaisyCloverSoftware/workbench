package core

import (
	"strings"
	"testing"
	"time"
)

func TestDelegateOperationRejectsChatGPTMCP(t *testing.T) {
	e := &Engine{}
	_, err := e.DelegateOperation("chatgpt-mcp", "Restart a service", "")
	if err == nil || !strings.Contains(err.Error(), "implicit OpenClaw delegation is disabled") {
		t.Fatalf("ChatGPT operation delegation was not refused: %v", err)
	}
}

func TestRetireLegacyChatGPTOperationTasksCancelsOnlyLegacyChatGPTWork(t *testing.T) {
	now := time.Date(2026, 8, 20, 3, 20, 0, 0, time.UTC)
	retryAt := now.Add(5 * time.Minute)
	st := State{Tasks: []Task{
		{
			ID:           "task-legacy-running",
			Origin:       "chatgpt-mcp",
			Mode:         TaskModeOperations,
			Status:       TaskRunning,
			ProviderID:   "openclaw",
			RouteReason:  "included subscription worker before scarce Work",
			ConsumesWork: false,
		},
		{
			ID:         "task-legacy-retry",
			Origin:     "chatgpt-mcp",
			Mode:       TaskModeOperations,
			Status:     TaskWaitingRetry,
			ProviderID: "openclaw",
			RetryAt:    &retryAt,
		},
		{
			ID:         "task-manual-operator",
			Origin:     "workbench-ui",
			Mode:       TaskModeOperations,
			Status:     TaskRunning,
			ProviderID: "openclaw",
		},
		{
			ID:       "task-legacy-complete",
			Origin:   "chatgpt-mcp",
			Mode:     TaskModeOperations,
			Status:   TaskCompleted,
			FinishedAt: timePointer(now.Add(-time.Minute)),
		},
	}}

	if !retireLegacyChatGPTOperationTasks(&st, now) {
		t.Fatal("expected legacy ChatGPT operations to be retired")
	}

	for _, index := range []int{0, 1} {
		got := st.Tasks[index]
		if got.Status != TaskCancelled || got.ProviderID != "" || got.RouteReason != "" || got.RetryAt != nil || got.FinishedAt == nil {
			t.Fatalf("legacy task was not fully retired: %#v", got)
		}
		if len(got.Attempts) == 0 || !strings.Contains(got.Attempts[len(got.Attempts)-1], "will not be resumed") {
			t.Fatalf("legacy task retirement reason missing: %#v", got.Attempts)
		}
	}

	manual := st.Tasks[2]
	if manual.Status != TaskRunning || manual.ProviderID != "openclaw" {
		t.Fatalf("manual OpenClaw operation was incorrectly retired: %#v", manual)
	}
	if st.Tasks[3].Status != TaskCompleted {
		t.Fatalf("terminal task was changed: %#v", st.Tasks[3])
	}
}
