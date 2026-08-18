package core

import (
	"errors"
	"strings"
	"time"
)

// DelegateOperation creates the Workbench lane that replaces manual
// ChatGPT→OpenClaw clipboard work. The ordinary Engine execution machinery still
// owns durability, retries, attention and reporting, while RunProviderIsolated
// enforces that only the cluster runner/OpenClaw may execute this task mode.
func (e *Engine) DelegateOperation(origin, intent, project string) (Task, error) {
	if strings.TrimSpace(intent) == "" {
		return Task{}, errors.New("tell Workbench what operational outcome you want")
	}
	project, err := canonicalProjectSelection(project)
	if err != nil {
		return Task{}, errors.New("choose a valid project folder first")
	}
	now := time.Now()
	t := Task{
		ID:          newID("task"),
		CreatedAt:   now,
		UpdatedAt:   now,
		Origin:      origin,
		Title:       TaskTitle(intent),
		Intent:      intent,
		ProjectPath: project,
		Mode:        TaskModeOperations,
		Status:      TaskQueued,
	}
	e.mu.Lock()
	touchProjectState(&e.state, project)
	e.state.Tasks = append([]Task{t}, e.state.Tasks...)
	st := cloneState(e.state)
	e.mu.Unlock()
	if err := e.store.Save(st); err != nil {
		return Task{}, err
	}
	e.notify()
	go e.execute(t.ID)
	return t, nil
}
