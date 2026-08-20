package core

import (
	"errors"
	"sort"
	"time"
)

func (e *Engine) wakeScheduler() {
	if e == nil || e.schedulerWake == nil {
		return
	}
	select {
	case e.schedulerWake <- struct{}{}:
	default:
	}
}

func (e *Engine) schedulerLoop() {
	for range e.schedulerWake {
		for {
			launch := e.schedulerDispatch()
			if len(launch) == 0 {
				break
			}
			for _, taskID := range launch {
				go e.execute(taskID)
			}
		}
	}
}

// schedulerDispatch owns the queued -> routing transition. A lane only receives
// another task when its execution capacity is free. This makes TaskQueued a
// durable, observable state instead of a transient label immediately followed
// by an unbounded goroutine.
func (e *Engine) schedulerDispatch() []string {
	e.mu.Lock()
	active := map[WorkLane]int{}
	for _, task := range e.state.Tasks {
		if task.Status == TaskRouting || task.Status == TaskRunning {
			active[TaskLane(task)]++
		}
	}

	queued := make([]int, 0)
	for i := range e.state.Tasks {
		if e.state.Tasks[i].Status == TaskQueued {
			queued = append(queued, i)
		}
	}
	sort.SliceStable(queued, func(a, b int) bool {
		left, right := e.state.Tasks[queued[a]], e.state.Tasks[queued[b]]
		lr, rr := priorityRank(DefaultTaskPriority(left)), priorityRank(DefaultTaskPriority(right))
		if lr != rr {
			return lr < rr
		}
		if !left.CreatedAt.Equal(right.CreatedAt) {
			return left.CreatedAt.Before(right.CreatedAt)
		}
		return left.ID < right.ID
	})

	launch := make([]string, 0)
	now := time.Now()
	for _, idx := range queued {
		task := &e.state.Tasks[idx]
		lane := TaskLane(*task)
		if schedulerLaneCapacity(lane) <= active[lane] {
			continue
		}
		task.Status = TaskRouting
		task.Progress = WorkProgress{Kind: ProgressIndeterminate, Phase: "Selecting executor"}
		task.UpdatedAt = now
		active[lane]++
		launch = append(launch, task.ID)
	}
	if len(launch) == 0 {
		e.mu.Unlock()
		return nil
	}
	st := cloneState(e.state)
	e.mu.Unlock()
	_ = e.store.Save(st)
	e.notify()
	return launch
}

func schedulerLaneCapacity(lane WorkLane) int {
	switch lane {
	case WorkLaneServerOps, WorkLaneCIBuilds, WorkLaneWindowsWorkstation, WorkLaneAIWorkers:
		return 1
	default:
		return 0
	}
}

func (e *Engine) SetTaskPriority(taskID string, priority WorkPriority) error {
	if priority != PriorityCritical && priority != PriorityHigh && priority != PriorityNormal && priority != PriorityLow {
		return errors.New("invalid task priority")
	}
	e.mu.Lock()
	i := e.taskIndexLocked(taskID)
	if i < 0 {
		e.mu.Unlock()
		return errors.New("task not found")
	}
	if e.state.Tasks[i].Status != TaskQueued {
		e.mu.Unlock()
		return errors.New("only queued tasks can be reprioritised")
	}
	e.state.Tasks[i].Priority = priority
	e.state.Tasks[i].UpdatedAt = time.Now()
	st := cloneState(e.state)
	e.mu.Unlock()
	if err := e.store.Save(st); err != nil {
		return err
	}
	e.notify()
	e.wakeScheduler()
	return nil
}
