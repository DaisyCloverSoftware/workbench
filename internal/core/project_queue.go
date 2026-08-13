package core

import "strings"

func projectTaskBlocks(status TaskStatus) bool {
	switch status {
	case TaskQueued, TaskRouting, TaskRunning, TaskNeedsAttention:
		return true
	default:
		return false
	}
}

// claimProjectTurn returns true only for the oldest non-terminal task in a
// project. Delegate may launch lightweight goroutines freely; later tasks exit
// immediately and remain queued until the active task reaches a terminal state.
func (e *Engine) claimProjectTurn(taskID string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	idx := e.taskIndexLocked(taskID)
	if idx < 0 || !projectTaskBlocks(e.state.Tasks[idx].Status) {
		return false
	}
	project := strings.TrimSpace(e.state.Tasks[idx].ProjectPath)
	for i := len(e.state.Tasks) - 1; i >= 0; i-- {
		t := e.state.Tasks[i]
		if strings.TrimSpace(t.ProjectPath) == project && projectTaskBlocks(t.Status) {
			return t.ID == taskID
		}
	}
	return false
}

// startNextQueued starts the oldest waiting task only when no routing/running
// task or genuine-attention pause still owns the project.
func (e *Engine) startNextQueued(project string) {
	project = strings.TrimSpace(project)
	if project == "" {
		return
	}
	e.mu.RLock()
	for _, t := range e.state.Tasks {
		if strings.TrimSpace(t.ProjectPath) != project {
			continue
		}
		if t.Status == TaskRouting || t.Status == TaskRunning || t.Status == TaskNeedsAttention {
			e.mu.RUnlock()
			return
		}
	}
	var next string
	for i := len(e.state.Tasks) - 1; i >= 0; i-- {
		t := e.state.Tasks[i]
		if strings.TrimSpace(t.ProjectPath) == project && t.Status == TaskQueued {
			next = t.ID
			break
		}
	}
	e.mu.RUnlock()
	if next != "" {
		go e.execute(next)
	}
}
