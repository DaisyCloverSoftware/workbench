package core

import (
	"errors"
	"strings"
	"time"
)

const OpenClawExplicitAuthorizationPrefix = "[workbench:openclaw-owner-authorized]"

// DelegateOperation creates the autonomous OpenClaw operator lane only for an
// explicitly owner-authorized operation. Availability, difficulty, a direct
// allowlist miss, or the ordinary [workbench:operations] routing marker are not
// authorization. ChatGPT owns routine machine work through direct Workbench
// controls and reviewed operations scripts.
func (e *Engine) DelegateOperation(origin, intent, project string) (Task, error) {
	intent = strings.TrimSpace(intent)
	if intent == "" {
		return Task{}, errors.New("tell Workbench what operational outcome you want")
	}
	if !strings.HasPrefix(intent, OpenClawExplicitAuthorizationPrefix) {
		return Task{}, errors.New("OpenClaw is owner-opt-in only; explicit owner authorization naming OpenClaw is required")
	}
	intent = strings.TrimSpace(strings.TrimPrefix(intent, OpenClawExplicitAuthorizationPrefix))
	if intent == "" {
		return Task{}, errors.New("OpenClaw owner authorization must include an operational objective")
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
	e.wakeScheduler()
	return t, nil
}
