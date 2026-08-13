package core

import (
	"path/filepath"
	"strings"
)

// taskKnowledgeProject keeps durable memory keyed to the logical source
// repository even when a worker is executing inside a Workbench-owned detached
// task workspace. The workspace directory and its sibling metadata file share
// the same private deterministic key, so no extra task field or worker-visible
// control value is required.
func taskKnowledgeProject(task Task) string {
	project := strings.TrimSpace(task.ProjectPath)
	if project == "" || strings.TrimSpace(task.ID) == "" {
		return project
	}
	abs, err := filepath.Abs(project)
	if err != nil {
		return project
	}
	metadataPath := abs + ".json"
	ws, ok, err := loadTaskWorkspace(metadataPath)
	if err != nil || !ok {
		return project
	}
	if ws.TaskID != strings.TrimSpace(task.ID) {
		return project
	}
	workspaceAbs, err := filepath.Abs(ws.Workspace)
	if err != nil || filepath.Clean(workspaceAbs) != filepath.Clean(abs) {
		return project
	}
	if strings.TrimSpace(ws.Project) == "" {
		return project
	}
	return strings.TrimSpace(ws.Project)
}
