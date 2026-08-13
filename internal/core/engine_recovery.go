package core

import (
	"fmt"
	"strings"
	"time"
)

// recoverInterruptedTasks turns non-terminal work left behind by a previous
// Workbench process into queued work that the new Engine instance can resume.
// Human-attention and terminal states are deliberately left untouched.
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
			if previous == TaskRouting || previous == TaskRunning {
				detail := "Workbench restarted; resuming interrupted task"
				if provider := strings.TrimSpace(t.ProviderID); provider != "" {
					detail = fmt.Sprintf("Workbench restarted; resuming task interrupted while %s on %s", previous, provider)
				} else {
					detail = fmt.Sprintf("Workbench restarted; resuming task interrupted while %s", previous)
				}
				t.Attempts = append(t.Attempts, detail)
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
