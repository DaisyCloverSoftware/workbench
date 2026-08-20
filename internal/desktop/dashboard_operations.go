package desktop

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

type DashboardOperationsSnapshot struct {
	Items       []core.WorkItem
	ByLane      map[core.WorkLane][]core.WorkItem
	Queued      int
	Running     int
	Waiting     int
	NeedsHuman  int
}

func BuildDashboardOperationsSnapshot(eng *core.Engine) DashboardOperationsSnapshot {
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
		out.Items = append(out.Items, item)
		out.ByLane[lane] = append(out.ByLane[lane], item)
		switch task.Status {
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
	for lane := range out.ByLane {
		sort.SliceStable(out.ByLane[lane], func(i, j int) bool {
			a, b := out.ByLane[lane][i], out.ByLane[lane][j]
			if a.Priority != b.Priority {
				return a.Priority < b.Priority
			}
			if a.QueuePosition > 0 && b.QueuePosition > 0 && a.QueuePosition != b.QueuePosition {
				return a.QueuePosition < b.QueuePosition
			}
			return a.UpdatedAt.After(b.UpdatedAt)
		})
	}
	return out
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
