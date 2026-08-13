package core

import (
	"fmt"
	"strings"
	"time"
)

// ResumeInterruptedTasks restarts durable queued/routing/running tasks after a
// headless Workbench instance has successfully acquired its MCP listener. Tasks
// already waiting for a human decision and terminal tasks remain untouched.
func (e *Engine) ResumeInterruptedTasks() error {
	e.mu.Lock()
	ids := recoverInterruptedTasks(&e.state)
	st := cloneState(e.state)
	e.mu.Unlock()
	if len(ids) == 0 {
		return nil
	}
	if err := e.store.Save(st); err != nil {
		return err
	}
	e.notify()
	for _, id := range ids {
		go e.execute(id)
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
			t.FinishedAt = nil
			t.UpdatedAt = now
			ids = append(ids, t.ID)
		}
	}
	return ids
}
