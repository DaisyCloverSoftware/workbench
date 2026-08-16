package desktop

import (
	"sort"
	"strings"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

type DashboardSnapshot struct {
	GeneratedAt      time.Time
	Summary          core.TaskDashboardSummary
	ProjectCount     int
	ProviderReady    int
	ProviderTotal    int
	SuccessRate      int
	RunnerConfigured bool
	RecentActivity   []DashboardActivityItem
	ActiveTasks      []DashboardTaskItem
	Projects         []DashboardProjectItem
	Providers        []DashboardProviderItem
}

type DashboardActivityItem struct {
	TaskID      string
	Title       string
	Detail      string
	Status      core.TaskStatus
	StatusLabel string
	UpdatedAt   time.Time
}

type DashboardTaskItem struct {
	TaskID      string
	Title       string
	Provider    string
	Status      core.TaskStatus
	StatusLabel string
	NextAction  string
	NeedsHuman  bool
	RetryAt     *time.Time
}

type DashboardProjectItem struct {
	ID      string
	Name    string
	Path    string
	Pinned  bool
	Summary core.TaskDashboardSummary
}

type DashboardProviderItem struct {
	ID          string
	Name        string
	Status      string
	Cost        core.CostClass
	Ready       bool
	Cooling     bool
	CurrentTask string
}

func BuildDashboardSnapshot(eng *core.Engine) DashboardSnapshot {
	now := time.Now().UTC()
	if eng == nil {
		return DashboardSnapshot{GeneratedAt: now}
	}
	st := eng.State()
	providers := eng.Providers()
	projects := eng.Projects()
	allTasks := append([]core.Task(nil), st.Tasks...)

	snapshot := DashboardSnapshot{
		GeneratedAt:      now,
		Summary:          core.SummarizeTasks(allTasks),
		ProjectCount:     len(projects),
		RunnerConfigured: strings.TrimSpace(st.Preferences.OpenClawSSHHost) != "",
	}
	terminal := snapshot.Summary.Completed + snapshot.Summary.Failed
	if terminal > 0 {
		snapshot.SuccessRate = int((float64(snapshot.Summary.Completed)/float64(terminal))*100 + 0.5)
	}

	activeByProvider := map[string]string{}
	for _, task := range allTasks {
		if isDashboardActiveStatus(task.Status) && strings.TrimSpace(task.ProviderID) != "" {
			if _, exists := activeByProvider[task.ProviderID]; !exists {
				activeByProvider[task.ProviderID] = task.Title
			}
		}
	}
	for _, provider := range providers {
		// Provider health is an execution-capacity panel. Chat/MCP is reported in
		// System status instead of being counted as a coding worker simply because
		// the local bridge listener exists.
		if provider.ID != "workbench-runner" && !core.IsCodingWorkerProvider(provider) {
			continue
		}
		ready := false
		if provider.ID == "workbench-runner" {
			ready = snapshot.RunnerConfigured && provider.Installed && provider.Authenticated && strings.TrimSpace(provider.Command) != ""
		} else {
			ready = core.ProviderReadyForCoding(provider)
		}
		snapshot.ProviderTotal++
		if ready {
			snapshot.ProviderReady++
		}
		snapshot.Providers = append(snapshot.Providers, DashboardProviderItem{
			ID:          provider.ID,
			Name:        provider.Name,
			Status:      strings.TrimSpace(provider.Status),
			Cost:        provider.Cost,
			Ready:       ready,
			Cooling:     strings.Contains(strings.ToLower(provider.Status), "cooldown"),
			CurrentTask: activeByProvider[provider.ID],
		})
	}
	sort.SliceStable(snapshot.Providers, func(i, j int) bool {
		if snapshot.Providers[i].Ready != snapshot.Providers[j].Ready {
			return snapshot.Providers[i].Ready
		}
		if snapshot.Providers[i].Cooling != snapshot.Providers[j].Cooling {
			return !snapshot.Providers[i].Cooling
		}
		return strings.ToLower(snapshot.Providers[i].Name) < strings.ToLower(snapshot.Providers[j].Name)
	})

	for _, project := range projects {
		snapshot.Projects = append(snapshot.Projects, DashboardProjectItem{
			ID:      project.ID,
			Name:    project.Name,
			Path:    project.Path,
			Pinned:  project.Pinned,
			Summary: core.SummarizeTasks(eng.TasksForProject(project.ID)),
		})
	}

	sort.SliceStable(allTasks, func(i, j int) bool {
		return allTasks[i].UpdatedAt.After(allTasks[j].UpdatedAt)
	})
	for _, task := range allTasks {
		presentation := core.PresentTask(task)
		label := dashboardStatusLabel(task.Status, presentation.StatusLabel)
		if len(snapshot.RecentActivity) < 6 {
			detail := presentation.NextAction
			if len(task.Attempts) > 0 {
				detail = task.Attempts[len(task.Attempts)-1]
			}
			snapshot.RecentActivity = append(snapshot.RecentActivity, DashboardActivityItem{
				TaskID:      task.ID,
				Title:       task.Title,
				Detail:      strings.TrimSpace(detail),
				Status:      task.Status,
				StatusLabel: label,
				UpdatedAt:   task.UpdatedAt,
			})
		}
		if isDashboardActiveStatus(task.Status) && len(snapshot.ActiveTasks) < 6 {
			provider := strings.TrimSpace(task.ProviderID)
			if provider == "" {
				provider = presentation.ProviderLabel
			}
			snapshot.ActiveTasks = append(snapshot.ActiveTasks, DashboardTaskItem{
				TaskID:      task.ID,
				Title:       task.Title,
				Provider:    provider,
				Status:      task.Status,
				StatusLabel: label,
				NextAction:  presentation.NextAction,
				NeedsHuman:  presentation.NeedsHuman,
				RetryAt:     task.RetryAt,
			})
		}
	}
	return snapshot
}

func isDashboardActiveStatus(status core.TaskStatus) bool {
	switch status {
	case core.TaskQueued, core.TaskRouting, core.TaskRunning, core.TaskWaitingRetry, core.TaskWaitingDependency, core.TaskNeedsAttention:
		return true
	default:
		return false
	}
}

func dashboardStatusLabel(status core.TaskStatus, fallback string) string {
	switch status {
	case core.TaskQueued:
		return "Queued"
	case core.TaskRouting:
		return "Routing"
	case core.TaskRunning:
		return "Working"
	case core.TaskWaitingRetry:
		return "Waiting"
	case core.TaskWaitingDependency:
		return "Waiting on dependency"
	case core.TaskNeedsAttention:
		return "Needs you"
	case core.TaskCompleted:
		return "Ready"
	case core.TaskFailed:
		return "Failed"
	case core.TaskCancelled:
		return "Cancelled"
	default:
		return strings.TrimSpace(fallback)
	}
}
