package core

import (
	"fmt"
	"strings"
	"time"
)

type taskRetrySchedule struct {
	TaskID  string
	RetryAt time.Time
}

// ResumeInterruptedTasks restarts durable queued/routing/running tasks after a
// headless Workbench instance has successfully acquired its MCP listener. Tasks
// already waiting for a human decision and terminal tasks remain untouched.
// Future automatic retries keep their persisted deadline and are re-armed rather
// than being forced to run immediately after every Workbench restart.
func (e *Engine) ResumeInterruptedTasks() error {
	e.mu.Lock()
	ids := recoverInterruptedTasks(&e.state)
	retryNow, retryLater := recoverWaitingRetryTasks(&e.state, time.Now().UTC())
	ids = append(ids, retryNow...)
	stateChanged := len(ids) > 0
	st := cloneState(e.state)
	e.mu.Unlock()

	if stateChanged {
		if err := e.store.Save(st); err != nil {
			return err
		}
		e.notify()
	}
	for _, id := range ids {
		go e.execute(id)
	}
	for _, retry := range retryLater {
		go e.scheduleAutomaticRetry(retry.TaskID, retry.RetryAt)
	}
	return nil
}

func recoverInterruptedTasks(st *State) []string {
	if st == nil {
		return nil
	}
	now := time.Now().UTC()
	ids := make([]string, 0)
	for i := range st.Tasks {
		t := &st.Tasks[i]
		switch t.Status {
		case TaskQueued, TaskRouting, TaskRunning:
			previous := t.Status
			if previous != TaskQueued {
				reason := fmt.Sprintf("Workbench restarted; resuming task interrupted while %s", previous)
				if provider := strings.TrimSpace(t.ProviderID); provider != "" {
					reason += " on " + provider
				}
				t.Attempts = append(t.Attempts, reason)
			}
			t.Status = TaskQueued
			t.ProviderID = ""
			t.RouteReason = ""
			t.ConsumesWork = false
			t.RetryAt = nil
			t.FinishedAt = nil
			t.UpdatedAt = now
			ids = append(ids, t.ID)
		}
	}
	return ids
}

func recoverWaitingRetryTasks(st *State, now time.Time) ([]string, []taskRetrySchedule) {
	if st == nil {
		return nil, nil
	}
	now = now.UTC()
	var retryNow []string
	var retryLater []taskRetrySchedule
	for i := range st.Tasks {
		t := &st.Tasks[i]
		if t.Status != TaskWaitingRetry {
			continue
		}
		if t.RetryAt != nil && t.RetryAt.After(now) {
			retryLater = append(retryLater, taskRetrySchedule{TaskID: t.ID, RetryAt: t.RetryAt.UTC()})
			continue
		}
		t.Status = TaskQueued
		t.ProviderID = ""
		t.RouteReason = ""
		t.ConsumesWork = false
		t.RetryAt = nil
		t.FinishedAt = nil
		t.UpdatedAt = now
		t.Attempts = append(t.Attempts, "Workbench restarted after the automatic retry deadline; retrying now")
		retryNow = append(retryNow, t.ID)
	}
	return retryNow, retryLater
}
