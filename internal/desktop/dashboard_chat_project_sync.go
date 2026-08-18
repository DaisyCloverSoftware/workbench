package desktop

import (
	"strings"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

const chatProjectAutoRegisterWindow = 45 * time.Minute

// registerActiveChatProjects makes the desktop project rail follow the way this
// product is actually used: the human talks to ChatGPT and Workbench supplies
// the hands. A project should therefore become a first-class desktop project as
// soon as ChatGPT has genuinely used Workbench on it; the user should not have
// to return to Workbench and import the same repository manually.
//
// We only register refs that are both in the runner's current project inventory
// and in recent private-relay activity. Old relay history cannot resurrect a
// removed project, and unrelated runner checkouts do not automatically clutter
// the desktop merely because they exist on disk.
func registerActiveChatProjects(eng *core.Engine, inventory []core.RunnerProjectInfo, activity []core.RunnerChatActivityInfo, now time.Time) (int, error) {
	if eng == nil || len(inventory) == 0 || len(activity) == 0 {
		return 0, nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	cutoff := now.UTC().Add(-chatProjectAutoRegisterWindow)

	available := make(map[string]core.RunnerProjectInfo, len(inventory))
	for _, project := range inventory {
		ref, ok := core.NormalizeRunnerProjectReference(project.Ref)
		if !ok {
			continue
		}
		name, ok := core.RunnerProjectName(ref)
		if !ok || !strings.EqualFold(strings.TrimSpace(project.Name), name) {
			continue
		}
		project.Ref = ref
		project.Name = name
		available[strings.ToLower(ref)] = project
	}

	existing := map[string]bool{}
	for _, project := range eng.Projects() {
		if ref, ok := core.NormalizeRunnerProjectReference(project.Path); ok {
			existing[strings.ToLower(ref)] = true
		}
	}

	wanted := map[string]core.RunnerProjectInfo{}
	for _, event := range activity {
		if event.UpdatedAt.IsZero() || event.UpdatedAt.UTC().Before(cutoff) {
			continue
		}
		ref, ok := core.NormalizeRunnerProjectReference(event.ProjectRef)
		if !ok {
			continue
		}
		key := strings.ToLower(ref)
		if existing[key] {
			continue
		}
		if project, ok := available[key]; ok {
			wanted[key] = project
		}
	}
	if len(wanted) == 0 {
		return 0, nil
	}

	projects := make([]core.RunnerProjectInfo, 0, len(wanted))
	for _, project := range inventory {
		ref, ok := core.NormalizeRunnerProjectReference(project.Ref)
		if !ok {
			continue
		}
		if candidate, ok := wanted[strings.ToLower(ref)]; ok {
			projects = append(projects, candidate)
			delete(wanted, strings.ToLower(ref))
		}
	}
	return eng.RegisterRunnerProjects(projects)
}
