package desktop

import (
	"strings"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

// filterRunnerChatActivityToInventory prevents old relay history for deleted,
// review-worktree or maintenance checkouts from becoming user-facing dashboard
// work after the runner has stopped advertising that checkout as a project.
func filterRunnerChatActivityToInventory(inventory []core.RunnerProjectInfo, activity []core.RunnerChatActivityInfo) []core.RunnerChatActivityInfo {
	available := make(map[string]bool, len(inventory))
	for _, project := range inventory {
		if ref, ok := core.NormalizeRunnerProjectReference(project.Ref); ok {
			available[strings.ToLower(ref)] = true
		}
	}
	if len(available) == 0 || len(activity) == 0 {
		return nil
	}
	out := make([]core.RunnerChatActivityInfo, 0, len(activity))
	for _, event := range activity {
		ref, ok := core.NormalizeRunnerProjectReference(event.ProjectRef)
		if !ok || !available[strings.ToLower(ref)] {
			continue
		}
		event.ProjectRef = ref
		out = append(out, event)
	}
	return out
}

// pruneUnavailableRunnerProjects removes only empty, unpinned runner entries
// that no longer exist in a successful non-empty runner inventory. Durable task
// history and explicitly pinned projects are retained. Refusing to prune from
// an empty inventory makes transient discovery degradation non-destructive.
func pruneUnavailableRunnerProjects(eng *core.Engine, inventory []core.RunnerProjectInfo) int {
	if eng == nil || len(inventory) == 0 {
		return 0
	}
	available := make(map[string]bool, len(inventory))
	for _, project := range inventory {
		if ref, ok := core.NormalizeRunnerProjectReference(project.Ref); ok {
			available[strings.ToLower(ref)] = true
		}
	}
	if len(available) == 0 {
		return 0
	}
	removed := 0
	for _, project := range eng.Projects() {
		ref, ok := core.NormalizeRunnerProjectReference(project.Path)
		if !ok || available[strings.ToLower(ref)] || project.Pinned {
			continue
		}
		if len(eng.TasksForProject(project.ID)) != 0 {
			continue
		}
		if err := eng.RemoveProject(project.ID); err == nil {
			removed++
		}
	}
	return removed
}
