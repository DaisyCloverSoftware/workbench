package core

import (
	"errors"
	"strings"
)

const manualQueueRankStep int64 = 1000

// ReorderQueuedTasks persists an exact drag/reorder result for one task lane and
// one priority group. It refuses partial lists so a stale UI cannot silently
// drop or reorder jobs it did not know existed.
func (e *Engine) ReorderQueuedTasks(lane WorkLane, priority WorkPriority, orderedTaskIDs []string) error {
	if lane != WorkLaneServerOperations && lane != WorkLaneAIWorkers {
		return errors.New("this task queue lane does not support manual ordering")
	}
	if !IsWorkPriority(priority) {
		return errors.New("queue priority must be Critical, High, Normal, or Low")
	}
	priority = NormalizeWorkPriority(priority)
	if len(orderedTaskIDs) > 10000 {
		return errors.New("queue reorder request is too large")
	}

	ordered := make([]string, len(orderedTaskIDs))
	seen := make(map[string]struct{}, len(orderedTaskIDs))
	for i, raw := range orderedTaskIDs {
		id := strings.TrimSpace(raw)
		if id == "" {
			return errors.New("queue reorder contains an empty task id")
		}
		if _, exists := seen[id]; exists {
			return errors.New("queue reorder contains a duplicate task id")
		}
		seen[id] = struct{}{}
		ordered[i] = id
	}

	e.mu.Lock()
	projects := make(map[string]Project, len(e.state.Projects))
	for _, project := range e.state.Projects {
		projects[normalizeProjectPath(project.Path)] = project
	}
	eligible := make(map[string]int)
	for i, task := range e.state.Tasks {
		if task.Status != TaskQueued {
			continue
		}
		project := projects[normalizeProjectPath(task.ProjectPath)]
		item := WorkItemFromTask(task, project)
		if item.Location.Lane == lane && item.Priority == priority {
			eligible[task.ID] = i
		}
	}
	if len(eligible) != len(ordered) {
		e.mu.Unlock()
		return errors.New("queue changed; refresh before reordering")
	}
	for _, id := range ordered {
		if _, ok := eligible[id]; !ok {
			e.mu.Unlock()
			return errors.New("queue changed; refresh before reordering")
		}
	}
	for position, id := range ordered {
		index := eligible[id]
		e.state.Tasks[index].QueueRank = int64(position+1) * manualQueueRankStep
	}
	st := cloneState(e.state)
	e.mu.Unlock()
	if err := e.store.Save(st); err != nil {
		return err
	}
	e.notify()
	return nil
}
