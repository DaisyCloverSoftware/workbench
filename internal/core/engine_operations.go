package core

import (
	"errors"
	"strings"
	"time"
)

// DelegateOperation creates the optional autonomous operator lane used by
// Workbench-local/manual callers when direct structured machine controls cannot
// express the required operation. ChatGPT itself owns bounded machine work via
// inspect_machine, run_machine_command and committed operations controls, so a
// chatgpt-mcp caller must never silently turn routine work into an OpenClaw
// session. This makes that boundary executable rather than advisory.
func (e *Engine) DelegateOperation(origin, intent, project string) (Task, error) {
	if strings.TrimSpace(intent) == "" {
		return Task{}, errors.New("tell Workbench what operational outcome you want")
	}
	if strings.EqualFold(strings.TrimSpace(origin), "chatgpt-mcp") {
		return Task{}, errors.New("ChatGPT-originated machine operations must use direct Workbench controls; implicit OpenClaw delegation is disabled")
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
