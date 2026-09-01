package core

import (
	"strings"
	"testing"
	"time"
)

func TestDelegateOperationRejectsImplicitOpenClawForEveryOrigin(t *testing.T) {
	for _, origin := range []string{"chatgpt-mcp", "workbench-ui", "relay", "manual"} {
		e := &Engine{}
		_, err := e.DelegateOperation(origin, RelayOperationsIntentPrefix+" restart a service", "")
		if err == nil || !strings.Contains(err.Error(), "explicit owner authorization naming OpenClaw is required") {
			t.Fatalf("origin %q implicit OpenClaw delegation was not refused: %v", origin, err)
		}
	}
}

func TestDelegateOperationAcceptsExplicitOwnerOpenClawAuthorization(t *testing.T) {
	// Project validation occurs only after the authorization gate. Reaching that
	// validation proves the explicit authorization was accepted rather than the
	// operation being rejected as an implicit fallback.
	e := &Engine{}
	_, err := e.DelegateOperation("chatgpt-mcp", OpenClawExplicitAuthorizationPrefix+" investigate the runtime", "")
	if err == nil || strings.Contains(err.Error(), "explicit owner authorization") {
		t.Fatalf("explicit OpenClaw authorization did not cross the authorization gate: %v", err)
	}
}

func TestRetireLegacyChatGPTOperationTasksCancelsOnlyLegacyChatGPTWork(t *testing.T) {
	now := time.Date(2026, 8, 20, 3, 20, 0, 0, time.UTC); retryAt := now.Add(5*time.Minute)
	st := State{Tasks: []Task{{ID:"task-legacy-running",Origin:"chatgpt-mcp",Mode:TaskModeOperations,Status:TaskRunning,ProviderID:"openclaw",RouteReason:"legacy",ConsumesWork:false},{ID:"task-legacy-retry",Origin:"chatgpt-mcp",Mode:TaskModeOperations,Status:TaskWaitingRetry,ProviderID:"openclaw",RetryAt:&retryAt},{ID:"task-manual-operator",Origin:"workbench-ui",Mode:TaskModeOperations,Status:TaskRunning,ProviderID:"openclaw"},{ID:"task-legacy-complete",Origin:"chatgpt-mcp",Mode:TaskModeOperations,Status:TaskCompleted,FinishedAt:timePointer(now.Add(-time.Minute))}}}
	if !retireLegacyChatGPTOperationTasks(&st, now){t.Fatal("expected legacy ChatGPT operations to be retired")}
	for _, index := range []int{0,1}{got:=st.Tasks[index]; if got.Status!=TaskCancelled||got.ProviderID!=""||got.RouteReason!=""||got.RetryAt!=nil||got.FinishedAt==nil{t.Fatalf("legacy task was not fully retired: %#v",got)}}
	manual:=st.Tasks[2]; if manual.Status!=TaskRunning||manual.ProviderID!="openclaw"{t.Fatalf("manual OpenClaw operation was incorrectly retired: %#v",manual)}
}
