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

type taskDependencySchedule struct {
	TaskID  string
	CheckAt time.Time
}

// ResumeInterruptedTasks restarts durable queued/routing/running tasks after a
// headless Workbench instance has successfully acquired its MCP listener. Tasks
// already waiting for a human decision and terminal tasks remain untouched.
// Future automatic retries and external dependency watches keep their persisted
// deadlines and are re-armed rather than being forced to run immediately after
// every Workbench restart.
func (e *Engine) ResumeInterruptedTasks() error {
	e.mu.Lock()
	now := time.Now().UTC()
	retiredLegacyChatGPT := retireLegacyChatGPTOperationTasks(&e.state, now)
	ids := recoverInterruptedTasks(&e.state)
	retryNow, retryLater := recoverWaitingRetryTasks(&e.state, now)
	ids = append(ids, retryNow...)
	dependencyLater, dependencyChanged := recoverWaitingDependencyTasks(&e.state, now)
	stateChanged := retiredLegacyChatGPT || len(ids) > 0 || dependencyChanged
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
	for _, dependency := range dependencyLater {
		go e.scheduleTaskDependencyCheck(dependency.TaskID, dependency.CheckAt)
	}
	return nil
}

// retireLegacyChatGPTOperationTasks is the migration boundary from the earlier
// ChatGPT→OpenClaw operations design to direct Workbench machine controls.
// Existing non-terminal tasks created by the old chatgpt-mcp operator path must
// not be resurrected after an upgrade/restart. Their history remains visible,
// but Workbench cancels them and clears execution state instead of opening a new
// OpenClaw session. Manual/local operations from other origins are unaffected.
func retireLegacyChatGPTOperationTasks(st *State, now time.Time) bool {
	if st == nil {
		return false
	}
	now = now.UTC()
	changed := false
	for i := range st.Tasks {
		t := &st.Tasks[i]
		if !strings.EqualFold(strings.TrimSpace(t.Origin), "chatgpt-mcp") || !IsOperationsTask(*t) {
			continue
		}
		switch t.Status {
		case TaskCompleted, TaskFailed, TaskCancelled:
			continue
		}
		t.Status = TaskCancelled
		t.ProviderID = ""
		t.RouteReason = ""
		t.ConsumesWork = false
		t.RetryAt = nil
		t.Dependency = nil
		t.AttentionQuestion = ""
		t.FinishedAt = timePointer(now)
		t.UpdatedAt = now
		t.Attempts = append(t.Attempts, "Workbench retired this legacy ChatGPT→OpenClaw operation during the direct-control migration; it will not be resumed")
		changed = true
	}
	return changed
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
			t.Dependency = nil
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
		t.Dependency = nil
		t.FinishedAt = nil
		t.UpdatedAt = now
		t.Attempts = append(t.Attempts, "Workbench restarted after the automatic retry deadline; retrying now")
		retryNow = append(retryNow, t.ID)
	}
	return retryNow, retryLater
}

func recoverWaitingDependencyTasks(st *State, now time.Time) ([]taskDependencySchedule, bool) {
	if st == nil {
		return nil, false
	}
	now = now.UTC()
	var schedules []taskDependencySchedule
	changed := false
	for i := range st.Tasks {
		t := &st.Tasks[i]
		if t.Status != TaskWaitingDependency {
			continue
		}
		if t.Dependency == nil || t.Dependency.Kind == "" {
			t.Status = TaskFailed
			t.Error = "Workbench could not recover this task because its external dependency metadata is missing."
			t.RouteReason = ""
			t.ConsumesWork = false
			t.Dependency = nil
			t.UpdatedAt = now
			t.FinishedAt = timePointer(now)
			changed = true
			continue
		}
		checkAt := t.Dependency.NextCheckAt.UTC()
		if checkAt.IsZero() || !checkAt.After(now) {
			// Give the control plane a moment to finish starting instead of firing a
			// network/CLI dependency probe from inside startup recovery.
			checkAt = now.Add(time.Second)
			t.Dependency.NextCheckAt = checkAt
			t.UpdatedAt = now
			changed = true
		}
		schedules = append(schedules, taskDependencySchedule{TaskID: t.ID, CheckAt: checkAt})
	}
	return schedules, changed
}

func timePointer(value time.Time) *time.Time {
	value = value.UTC()
	return &value
}
