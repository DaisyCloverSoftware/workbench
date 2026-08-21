package desktop

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

type DashboardOperationsSnapshot struct {
	Items      []core.WorkItem
	ByLane     map[core.WorkLane][]core.WorkItem
	Queued     int
	Running    int
	Waiting    int
	NeedsHuman int
}

func BuildDashboardOperationsSnapshot(eng *core.Engine) DashboardOperationsSnapshot {
	if eng != nil {
		ensureRunnerChatActivityMonitor(eng)
	}
	return buildDashboardOperationsSnapshot(eng, runnerChatActivitySnapshot())
}

func buildDashboardOperationsSnapshot(eng *core.Engine, remote []core.RunnerChatActivityInfo) DashboardOperationsSnapshot {
	out := DashboardOperationsSnapshot{ByLane: map[core.WorkLane][]core.WorkItem{}}
	if eng == nil {
		return out
	}
	st := eng.State()
	positions := core.QueuePositions(st.Tasks)
	projectNames := map[string]string{}
	for _, project := range eng.Projects() {
		projectNames[strings.TrimSpace(project.Path)] = project.Name
	}
	seen := map[string]bool{}
	for _, task := range st.Tasks {
		if !isDashboardActiveStatus(task.Status) {
			continue
		}
		lane := core.TaskLane(task)
		item := core.WorkItem{
			ID:            task.ID,
			ProjectPath:   task.ProjectPath,
			ProjectName:   projectNames[strings.TrimSpace(task.ProjectPath)],
			Title:         task.Title,
			State:         task.Status,
			Priority:      core.DefaultTaskPriority(task),
			Lane:          lane,
			QueuePosition: positions[task.ID],
			Provider:      task.ProviderID,
			Progress:      core.TaskProgress(task),
			CreatedAt:     task.CreatedAt,
			StartedAt:     task.StartedAt,
			UpdatedAt:     task.UpdatedAt,
			NeedsHuman:    task.Status == core.TaskNeedsAttention,
		}
		if item.ProjectName == "" {
			item.ProjectName = chatActivityProjectName(task.ProjectPath)
		}
		if task.Dependency != nil {
			item.Dependency = dependencySummary(task)
		}
		if strings.TrimSpace(item.Provider) == "" {
			switch task.Status {
			case core.TaskQueued:
				item.Provider = "Scheduler"
			case core.TaskWaitingDependency:
				item.Provider = "Dependency watcher"
			default:
				item.Provider = "Routing"
			}
		}
		appendDashboardWorkItem(&out, item)
		seen[item.ID] = true
	}

	// The desktop engine owns only Windows-local durable tasks. Work initiated by
	// ChatGPT normally lives on the cluster relay/runner, so merge the runner's
	// privacy-safe authoritative activity cache into the same operations board.
	// The runner's Active decision is authoritative: ordinary ChatGPT safe-hands
	// actions are individually short-lived and therefore often have a completed
	// result while the surrounding ChatGPT work session is still active. Treat
	// those active leases as running board work rather than dropping them solely
	// because the latest individual action has completed.
	for _, event := range remote {
		if !event.ActiveKnown || !event.Active || seen[event.ID] {
			continue
		}
		status, ok := remoteActivityTaskStatus(event)
		if !ok {
			continue
		}
		lane := remoteActivityLane(event.Action, status)
		projectName := chatActivityProjectName(event.ProjectRef)
		if projectName == "" {
			projectName = "Workbench"
		}
		item := core.WorkItem{
			ID:          event.ID,
			ProjectPath: event.ProjectRef,
			ProjectName: projectName,
			Title:       remoteActivityTitle(event.Action),
			State:       status,
			Priority:    core.PriorityNormal,
			Lane:        lane,
			Provider:    remoteActivityProvider(event.Action),
			Progress: core.WorkProgress{
				Kind:  core.ProgressIndeterminate,
				Phase: remoteActivityPhase(event.Action, status),
			},
			UpdatedAt:  event.UpdatedAt,
			NeedsHuman: status == core.TaskNeedsAttention,
		}
		appendDashboardWorkItem(&out, item)
		seen[item.ID] = true
	}

	for lane := range out.ByLane {
		sort.SliceStable(out.ByLane[lane], func(i, j int) bool {
			a, b := out.ByLane[lane][i], out.ByLane[lane][j]
			if dashboardPriorityRank(a.Priority) != dashboardPriorityRank(b.Priority) {
				return dashboardPriorityRank(a.Priority) < dashboardPriorityRank(b.Priority)
			}
			if a.QueuePosition > 0 && b.QueuePosition > 0 && a.QueuePosition != b.QueuePosition {
				return a.QueuePosition < b.QueuePosition
			}
			return a.UpdatedAt.After(b.UpdatedAt)
		})
	}
	return out
}

func appendDashboardWorkItem(out *DashboardOperationsSnapshot, item core.WorkItem) {
	if out == nil {
		return
	}
	out.Items = append(out.Items, item)
	out.ByLane[item.Lane] = append(out.ByLane[item.Lane], item)
	switch item.State {
	case core.TaskQueued:
		out.Queued++
	case core.TaskRunning, core.TaskRouting:
		out.Running++
	case core.TaskWaitingDependency, core.TaskWaitingRetry:
		out.Waiting++
	case core.TaskNeedsAttention:
		out.NeedsHuman++
	}
}

func remoteActivityTaskStatus(event core.RunnerChatActivityInfo) (core.TaskStatus, bool) {
	switch strings.ToLower(strings.TrimSpace(event.State)) {
	case "running":
		return core.TaskRunning, true
	case "waiting":
		return core.TaskWaitingDependency, true
	case "needs_attention":
		return core.TaskNeedsAttention, true
	case "completed", "failed":
		// ActiveKnown+Active is the runner's bounded session decision. A completed
		// or failed safe-hands operation can therefore still represent a live
		// ChatGPT work session. Completed autonomous delegate_task events are never
		// marked active by the runner, so this does not resurrect finished workers.
		if event.ActiveKnown && event.Active {
			return core.TaskRunning, true
		}
	}
	return "", false
}

func remoteActivityLane(action string, status core.TaskStatus) core.WorkLane {
	if status == core.TaskNeedsAttention {
		return core.WorkLaneNeedsYou
	}
	if status == core.TaskWaitingDependency || status == core.TaskWaitingRetry {
		return core.WorkLaneWaiting
	}
	action = strings.ToLower(strings.TrimSpace(action))
	switch {
	case action == "delegate_task":
		return core.WorkLaneAIWorkers
	case strings.HasPrefix(action, "run_windows_") || action == "get_windows_host_job":
		return core.WorkLaneWindowsWorkstation
	case action == "run_safe_command":
		return core.WorkLaneCIBuilds
	case action == "run_operations_script", action == "run_machine_command", action == "inspect_machine", action == "inspect_machine_batch", action == "update_workbench":
		return core.WorkLaneServerOps
	default:
		return core.WorkLaneAIWorkers
	}
}

func remoteActivityTitle(action string) string {
	action = strings.ToLower(strings.TrimSpace(action))
	switch action {
	case "delegate_task":
		return "Autonomous task"
	case "run_windows_unreal_smoke":
		return "Unreal smoke"
	case "run_windows_blender_version":
		return "Blender host job"
	case "run_operations_script":
		return "Server operation"
	case "run_machine_command":
		return "Machine operation"
	case "inspect_machine", "inspect_machine_batch":
		return "Machine inspection"
	case "run_safe_command":
		return "Build / safe command"
	case "update_workbench":
		return "Workbench update"
	default:
		if action == "" {
			return "Workbench activity"
		}
		return strings.ReplaceAll(action, "_", " ")
	}
}

func remoteActivityProvider(action string) string {
	action = strings.ToLower(strings.TrimSpace(action))
	switch {
	case action == "delegate_task":
		return "OpenClaw / Workbench"
	case strings.HasPrefix(action, "run_windows_") || action == "get_windows_host_job":
		return "Windows host bridge"
	case action == "run_safe_command":
		return "Workbench runner"
	default:
		return "Workbench private relay"
	}
}

func remoteActivityPhase(action string, status core.TaskStatus) string {
	switch status {
	case core.TaskWaitingDependency:
		return "Waiting on dependency"
	case core.TaskNeedsAttention:
		return "Needs human decision"
	}
	return remoteActivityTitle(action)
}

func dashboardPriorityRank(priority core.WorkPriority) int {
	switch priority {
	case core.PriorityCritical:
		return 0
	case core.PriorityHigh:
		return 1
	case core.PriorityNormal:
		return 2
	case core.PriorityLow:
		return 3
	default:
		return 2
	}
}

func dependencySummary(task core.Task) string {
	if task.Dependency == nil {
		return ""
	}
	if task.Dependency.Kind == core.DependencyGitHubActions {
		if task.Dependency.RunID > 0 {
			return fmt.Sprintf("GitHub Actions run %d", task.Dependency.RunID)
		}
		return "GitHub Actions"
	}
	return strings.TrimSpace(task.Dependency.Reason)
}

func workItemElapsed(item core.WorkItem, now time.Time) string {
	start := item.CreatedAt
	if item.StartedAt != nil && !item.StartedAt.IsZero() {
		start = *item.StartedAt
	}
	if start.IsZero() {
		return ""
	}
	d := now.Sub(start)
	if d < 0 {
		d = 0
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh %02dm", int(d.Hours()), int(d.Minutes())%60)
}
