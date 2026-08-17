package core

import (
	"errors"
	"strings"
	"time"
)

// DelegateWithCloudModel creates an ordinary Workbench task with one explicit
// model preference that is consulted only if the existing provider hierarchy
// eventually reaches the OpenClaw cloud stage. It never bypasses local models,
// the Workbench runner, Claude Code, structured harnesses, cost policy or any
// other outer routing decision.
func (e *Engine) DelegateWithCloudModel(origin, intent, project, cloudModel string) (Task, error) {
	cloudModel = strings.TrimSpace(cloudModel)
	if cloudModel == "" {
		return e.Delegate(origin, intent, project)
	}
	validatedModel, err := normalizeOpenClawCloudModelRef(cloudModel)
	if err != nil {
		return Task{}, err
	}
	if strings.TrimSpace(intent) == "" {
		return Task{}, errors.New("tell Workbench what outcome you want")
	}
	if task, deferred, err := e.tryDelegateDeferredDependency(origin, intent, project); deferred || err != nil {
		return task, err
	}
	project, err = canonicalProjectSelection(project)
	if err != nil {
		return Task{}, errors.New("choose a valid project folder first")
	}
	now := time.Now()
	t := Task{
		ID:                 newID("task"),
		CreatedAt:          now,
		UpdatedAt:          now,
		Origin:             origin,
		Title:              TaskTitle(intent),
		Intent:             intent,
		ProjectPath:        project,
		Status:             TaskQueued,
		CloudModelOverride: validatedModel,
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
