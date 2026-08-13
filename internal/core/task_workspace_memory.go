package core

import (
	"path/filepath"
	"strings"
)

// taskKnowledgeProject returns the durable logical project for a worker task.
// A Workbench task workspace has a sibling private metadata file with the same
// deterministic cache key, so the worker can execute in isolation without
// fragmenting project memory under a transient worktree path.
func taskKnowledgeProject(task Task) string {
	project := strings.TrimSpace(task.ProjectPath)
	if project == "" || strings.TrimSpace(task.ID) == "" {
		return project
	}
	abs, err := filepath.Abs(project)
	if err != nil {
		return project
	}
	ws, ok, err := loadTaskWorkspace(abs + ".json")
	if err != nil || !ok || ws.TaskID != strings.TrimSpace(task.ID) {
		return project
	}
	workspace, err := filepath.Abs(ws.Workspace)
	if err != nil || filepath.Clean(workspace) != filepath.Clean(abs) {
		return project
	}
	if strings.TrimSpace(ws.Project) == "" {
		return project
	}
	return strings.TrimSpace(ws.Project)
}
