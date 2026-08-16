package core

import "errors"

func (e *Engine) RegisterRunnerProjects(projects []RunnerProjectInfo) (int, error) {
	if len(projects) == 0 {
		return 0, nil
	}
	e.mu.Lock()
	existing := make(map[string]bool, len(e.state.Projects))
	for _, project := range e.state.Projects {
		existing[project.ID] = true
	}
	added := 0
	firstID := ""
	for _, candidate := range projects {
		name, ok := RunnerProjectName(candidate.Ref)
		if !ok || name != candidate.Name {
			e.mu.Unlock()
			return 0, errors.New("runner returned an invalid project reference")
		}
		project, err := upsertProjectLocked(&e.state, candidate.Ref, false)
		if err != nil {
			e.mu.Unlock()
			return 0, err
		}
		if firstID == "" {
			firstID = project.ID
		}
		if !existing[project.ID] {
			existing[project.ID] = true
			added++
		}
	}
	if e.state.ActiveProjectID == "" && firstID != "" {
		e.state.ActiveProjectID = firstID
		mirrorActiveProject(&e.state)
	}
	st := cloneState(e.state)
	e.mu.Unlock()
	if err := e.store.Save(st); err != nil {
		return 0, err
	}
	e.notify()
	return added, nil
}
