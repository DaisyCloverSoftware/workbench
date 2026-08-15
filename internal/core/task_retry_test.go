package core

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestAutomaticProviderRetryEligibilityIsTransientAndLowCostOnly(t *testing.T) {
	future := time.Now().UTC().Add(5 * time.Minute)
	included := Provider{ID: "claude", Cost: CostIncluded}
	for _, reason := range []string{"quota or rate limit", "worker timed out", "worker temporarily unavailable"} {
		if !automaticProviderRetryEligible(included, ProviderHealth{Reason: reason, CooldownUntil: future}) {
			t.Fatalf("transient included failure %q was not auto-retry eligible", reason)
		}
	}
	for _, reason := range []string{"authentication unavailable", "worker tool permissions unavailable", "adapter or CLI mismatch"} {
		if automaticProviderRetryEligible(included, ProviderHealth{Reason: reason, CooldownUntil: future}) {
			t.Fatalf("operator/setup failure %q became auto-retry eligible", reason)
		}
	}
	for _, cost := range []CostClass{CostScarce, CostMetered} {
		if automaticProviderRetryEligible(Provider{ID: "expensive", Cost: cost}, ProviderHealth{Reason: "worker temporarily unavailable", CooldownUntil: future}) {
			t.Fatalf("costly provider %s became auto-retry eligible", cost)
		}
	}
}

func TestAutomaticRetryAtUsesActiveProviderCooldown(t *testing.T) {
	isolateProviderHealthCache(t)
	now := time.Date(2026, 8, 15, 17, 0, 0, 0, time.UTC)
	record, err := recordProviderRetryableFailureAt("claude", RunResult{Retryable: true}, errors.New("quota exceeded"), now)
	if err != nil {
		t.Fatal(err)
	}
	retryAt, ok := automaticRetryAtForCoolingProviders([]Provider{{ID: "claude", Cost: CostIncluded}}, now.Add(time.Second))
	if !ok || !retryAt.Equal(record.CooldownUntil) {
		t.Fatalf("retryAt=%v ok=%t want %v", retryAt, ok, record.CooldownUntil)
	}
	if _, ok := automaticRetryAtForCoolingProviders([]Provider{{ID: "claude", Cost: CostScarce}}, now.Add(time.Second)); ok {
		t.Fatal("scarce cooling provider scheduled an automatic task retry")
	}
}

func TestDeferAutomaticRetryIsBoundedAndCancellationDisarmsTimer(t *testing.T) {
	store, err := NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	st := DefaultState()
	st.Tasks = []Task{{ID: "task-retry", Status: TaskRunning}}
	if err := store.Save(st); err != nil {
		t.Fatal(err)
	}
	e := &Engine{store: store, state: st, cancel: map[string]context.CancelFunc{}}

	scheduled, err := e.deferAutomaticRetry("task-retry", time.Now().UTC().Add(time.Minute))
	if err != nil || !scheduled {
		t.Fatalf("scheduled=%t err=%v", scheduled, err)
	}
	task, ok := e.Task("task-retry")
	if !ok || task.Status != TaskWaitingRetry || task.AutoRetryCount != 1 || task.RetryAt == nil {
		t.Fatalf("task was not durably deferred: %#v", task)
	}
	persistedRetryAt := *task.RetryAt
	if err := e.Cancel("task-retry"); err != nil {
		t.Fatal(err)
	}
	// Simulate the exact persisted timer firing after cancellation. The state
	// check must ignore it rather than queueing/executing the task again.
	e.fireAutomaticRetry("task-retry", persistedRetryAt)
	task, _ = e.Task("task-retry")
	if task.Status != TaskCancelled || task.RetryAt != nil {
		t.Fatalf("cancelled retry timer resurrected task: %#v", task)
	}

	e.mu.Lock()
	e.state.Tasks[0].Status = TaskRunning
	e.state.Tasks[0].AutoRetryCount = maxAutomaticTaskRetries
	e.state.Tasks[0].FinishedAt = nil
	e.mu.Unlock()
	scheduled, err = e.deferAutomaticRetry("task-retry", time.Now().UTC().Add(time.Minute))
	if err != nil || scheduled {
		t.Fatalf("retry beyond cap scheduled=%t err=%v", scheduled, err)
	}
}
