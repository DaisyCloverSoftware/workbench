package mcp

import (
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func workspaceProjectSummaries(engine *core.Engine, activeProjectID string) []map[string]any {
	if engine == nil {
		return nil
	}
	projects := engine.Projects()
	out := make([]map[string]any, 0, len(projects))
	for _, project := range projects {
		summary := core.SummarizeTasks(engine.TasksForProject(project.ID))
		lastUsed := ""
		if !project.LastUsedAt.IsZero() {
			lastUsed = project.LastUsedAt.UTC().Format(time.RFC3339)
		}
		out = append(out, map[string]any{
			"id":           project.ID,
			"name":         project.Name,
			"path":         project.Path,
			"pinned":       project.Pinned,
			"active":       project.ID == activeProjectID,
			"last_used_at": lastUsed,
			"tasks": map[string]any{
				"active":      summary.Active,
				"needs_human": summary.NeedsHuman,
				"completed":   summary.Completed,
				"failed":      summary.Failed,
			},
		})
	}
	return out
}
