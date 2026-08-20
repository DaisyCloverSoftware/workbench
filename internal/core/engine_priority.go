package core

import (
	"errors"
	"strings"
)

// IsWorkPriority reports whether priority is one of the four explicit values a
// user can select. Empty is intentionally not explicit: on a task it means
// "inherit the project default" and on an older project it means Normal.
func IsWorkPriority(priority WorkPriority) bool {
	switch WorkPriority(strings.ToLower(strings.TrimSpace(string(priority)))) {
	case WorkPriorityCritical, WorkPriorityHigh, WorkPriorityNormal, WorkPriorityLow:
		return true
	default:
		return false
	}
}

// EffectiveTaskPriority applies the durable scheduling inheritance rule. A task
// override wins; otherwise the registered project's default is used; legacy or
// unspecified project defaults safely become Normal.
func EffectiveTaskPriority(task Task, project Project) WorkPriority {
	if IsWorkPriority(task.Priority) {
		return NormalizeWorkPriority(task.Priority)
	}
	return NormalizeWorkPriority(project.DefaultPriority)
}

// SetProjectPriority changes the default used by work in this project. Existing
// task-specific overrides remain untouched, while inheriting queued/waiting
// work immediately observes the new default when projected or scheduled.
func (e *Engine) SetProjectPriority(projectID string, priority WorkPriority) error {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return errors.New("project id is empty")
	}
	if !IsWorkPriority(priority) {
		return errors.New("project priority must be Critical, High, Normal, or Low")
	}
	priority = NormalizeWorkPriority(priority)

	e.mu.Lock()
	found := false
	projectPath := ""
	for i := range e.state.Projects {
		if e.state.Projects[i].ID == projectID {
			e.state.Projects[i].DefaultPriority = priority
			projectPath = e.state.Projects[i].Path
			found = true
			break
		}
	}
	if !found {
		e.mu.Unlock()
		return errors.New("project not found")
	}
	// A project-default change can move inheriting queued work into another
	// priority group. Clear any prior manual rank so stale ordering from the old
	// group cannot unexpectedly jump ahead in the new group.
	for i := range e.state.Tasks {
		if e.state.Tasks[i].Status == TaskQueued && sameProjectPath(e.state.Tasks[i].ProjectPath, projectPath) && !IsWorkPriority(e.state.Tasks[i].Priority) {
			e.state.Tasks[i].QueueRank = 0
		}
	}
	st := cloneState(e.state)
	e.mu.Unlock()
	if err := e.store.Save(st); err != nil {
		return err
	}
	e.notify()
	return nil
}

// SetTaskPriority stores an explicit per-job override. Passing an empty value
// clears the override so the task follows its project default again.
func (e *Engine) SetTaskPriority(taskID string, priority WorkPriority) error {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return errors.New("task id is empty")
	}
	if strings.TrimSpace(string(priority)) != "" && !IsWorkPriority(priority) {
		return errors.New("task priority must be Critical, High, Normal, Low, or empty to inherit")
	}
	if IsWorkPriority(priority) {
		priority = NormalizeWorkPriority(priority)
	} else {
		priority = ""
	}

	e.mu.Lock()
	i := e.taskIndexLocked(taskID)
	if i < 0 {
		e.mu.Unlock()
		return errors.New("task not found")
	}
	e.state.Tasks[i].Priority = priority
	// Queue rank only has meaning inside one priority group. Changing the
	// effective group returns the task to FIFO until the queue is reordered.
	e.state.Tasks[i].QueueRank = 0
	st := cloneState(e.state)
	e.mu.Unlock()
	if err := e.store.Save(st); err != nil {
		return err
	}
	e.notify()
	return nil
}
