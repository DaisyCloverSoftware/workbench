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
// that no longer exist in a successful runner inventory. Durable task history
// and explicitly pinned projects are retained. This cleans up auto-registered
// temporary worktrees without losing project records that contain real work.
func pruneUnavailableRunnerProjects(eng *core.Engine, inventory []core.RunnerProjectInfo) int {
	if eng == nil {
		return 0
	}
	available := make(map[string]bool, len(inventory))
	for _, project := range inventory {
		if ref, ok := core.NormalizeRunnerProjectReference(project.Ref); ok {
			available[strings.ToLower(ref)] = true
		}
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
