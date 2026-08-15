package core

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

func (e *Engine) Projects() []Project {
	e.mu.RLock()
	projects := sortedProjects(e.state.Projects)
	e.mu.RUnlock()
	return projects
}

func (e *Engine) ActiveProject() (Project, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	for _, p := range e.state.Projects {
		if p.ID == e.state.ActiveProjectID {
			return p, true
		}
	}
	return Project{}, false
}

// ProjectByPath resolves one registered project without changing human
// workspace selection. It is used by read-only/multi-project control surfaces
// that need the project's own metadata rather than the active-project mirror.
func (e *Engine) ProjectByPath(path string) (Project, bool) {
	path = normalizeProjectPath(path)
	if path == "" {
		return Project{}, false
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	for _, project := range e.state.Projects {
		if sameProjectPath(project.Path, path) {
			return project, true
		}
	}
	return Project{}, false
}

func (e *Engine) SelectProject(path string) (Project, error) {
	path, err := canonicalProjectSelection(path)
	if err != nil {
		return Project{}, err
	}
	e.mu.Lock()
	project, err := upsertProjectLocked(&e.state, path, true)
	if err != nil {
		e.mu.Unlock()
		return Project{}, err
	}
	e.state.ActiveProjectID = project.ID
	mirrorActiveProject(&e.state)
	st := cloneState(e.state)
	e.mu.Unlock()
	if err := e.store.Save(st); err != nil {
		return Project{}, err
	}
	e.notify()
	return project, nil
}

func (e *Engine) SetProjectPinned(projectID string, pinned bool) error {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return errors.New("project id is empty")
	}
	e.mu.Lock()
	found := false
	for i := range e.state.Projects {
		if e.state.Projects[i].ID == projectID {
			e.state.Projects[i].Pinned = pinned
			found = true
			break
		}
	}
	if !found {
		e.mu.Unlock()
		return errors.New("project not found")
	}
	st := cloneState(e.state)
	e.mu.Unlock()
	if err := e.store.Save(st); err != nil {
		return err
	}
	e.notify()
	return nil
}

func (e *Engine) RenameProject(projectID, name string) error {
	projectID = strings.TrimSpace(projectID)
	name = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(name, "\r", " "), "\n", " "))
	if projectID == "" {
		return errors.New("project id is empty")
	}
	if name == "" {
		return errors.New("project name is empty")
	}
	if len(name) > 100 {
		return errors.New("project name is too long")
	}
	e.mu.Lock()
	found := false
	for i := range e.state.Projects {
		if e.state.Projects[i].ID == projectID {
			e.state.Projects[i].Name = name
			found = true
			break
		}
	}
	if !found {
		e.mu.Unlock()
		return errors.New("project not found")
	}
	st := cloneState(e.state)
	e.mu.Unlock()
	if err := e.store.Save(st); err != nil {
		return err
	}
	e.notify()
	return nil
}

func (e *Engine) RemoveProject(projectID string) error {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return errors.New("project id is empty")
	}
	e.mu.Lock()
	idx := -1
	for i := range e.state.Projects {
		if e.state.Projects[i].ID == projectID {
			idx = i
			break
		}
	}
	if idx < 0 {
		e.mu.Unlock()
		return errors.New("project not found")
	}
	projectPath := e.state.Projects[idx].Path
	unfinished := 0
	for _, task := range e.state.Tasks {
		if sameProjectPath(task.ProjectPath, projectPath) && taskBlocksProjectRemoval(task.Status) {
			unfinished++
		}
	}
	if unfinished > 0 {
		e.mu.Unlock()
		return fmt.Errorf("project has %d unfinished Workbench task(s); finish or cancel them before removing the project", unfinished)
	}
	e.state.Projects = append(e.state.Projects[:idx], e.state.Projects[idx+1:]...)
	if e.state.ActiveProjectID == projectID {
		e.state.ActiveProjectID = mostRecentProjectID(e.state.Projects)
	}
	mirrorActiveProject(&e.state)
	st := cloneState(e.state)
	e.mu.Unlock()
	if err := e.store.Save(st); err != nil {
		return err
	}
	e.notify()
	return nil
}

func taskBlocksProjectRemoval(status TaskStatus) bool {
	switch status {
	case TaskQueued, TaskRouting, TaskRunning, TaskWaitingRetry, TaskNeedsAttention:
		return true
	default:
		return false
	}
}

func (e *Engine) TasksForProject(projectID string) []Task {
	projectID = strings.TrimSpace(projectID)
	e.mu.RLock()
	var path string
	for _, p := range e.state.Projects {
		if p.ID == projectID {
			path = p.Path
			break
		}
	}
	if path == "" {
		e.mu.RUnlock()
		return nil
	}
	var tasks []Task
	for _, task := range e.state.Tasks {
		if sameProjectPath(task.ProjectPath, path) {
			copyTask := task
			copyTask.Attempts = append([]string(nil), task.Attempts...)
			copyTask.Review = cloneTaskReviewResult(task.Review)
			tasks = append(tasks, copyTask)
		}
	}
	e.mu.RUnlock()
	return tasks
}

// touchProjectState records that a project was used by a task without treating
// background delegation as a human navigation command. The existing active
// project remains active; only the first task in an otherwise empty registry
// establishes an active project. Explicit SelectProject remains the sole way to
// switch an already-established human workspace selection.
func touchProjectState(st *State, path string) {
	if st == nil || strings.TrimSpace(path) == "" {
		return
	}
	project, err := upsertProjectLocked(st, path, true)
	if err != nil {
		return
	}
	activeExists := false
	for _, existing := range st.Projects {
		if existing.ID == st.ActiveProjectID {
			activeExists = true
			break
		}
	}
	if !activeExists {
		st.ActiveProjectID = project.ID
	}
	mirrorActiveProject(st)
}

func saveProjectNotesState(st *State, projectPath, notes string) error {
	if st == nil {
		return errors.New("state is nil")
	}
	projectPath = normalizeProjectPath(projectPath)
	if projectPath == "" {
		return errors.New("choose a project first")
	}
	project, err := upsertProjectLocked(st, projectPath, false)
	if err != nil {
		return err
	}
	for i := range st.Projects {
		if st.Projects[i].ID == project.ID {
			st.Projects[i].Notes = notes
			if st.Projects[i].LastUsedAt.IsZero() {
				st.Projects[i].LastUsedAt = time.Now().UTC()
			}
			break
		}
	}
	activeExists := false
	for _, existing := range st.Projects {
		if existing.ID == st.ActiveProjectID {
			activeExists = true
			break
		}
	}
	if !activeExists {
		st.ActiveProjectID = project.ID
	}
	mirrorActiveProject(st)
	return nil
}
