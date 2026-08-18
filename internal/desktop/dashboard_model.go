package desktop

import (
	"sort"
	"strings"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

// Older runners did not report an explicit active decision, so retain the same
// bounded desktop fallback for compatibility. Current runners compute activity
// on the runner clock and set ActiveKnown instead.
const dashboardChatSessionWindow = 4 * time.Hour

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
	ensureRunnerChatActivityMonitor(eng)
	st := eng.State()
	providers := eng.Providers()
	projects := eng.Projects()
	allTasks := visibleTaskHistory(st.Tasks, false)

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
			Summary: core.SummarizeTasks(visibleTaskHistory(eng.TasksForProject(project.ID), false)),
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
	return applyRunnerChatActivity(snapshot, runnerChatActivitySnapshot(), now)
}

func applyRunnerChatActivity(snapshot DashboardSnapshot, activity []core.RunnerChatActivityInfo, now time.Time) DashboardSnapshot {
	if len(activity) == 0 {
		return snapshot
	}
	sort.SliceStable(activity, func(i, j int) bool { return activity[i].UpdatedAt.After(activity[j].UpdatedAt) })
	projectIndex := map[string]int{}
	for i := range snapshot.Projects {
		projectIndex[strings.TrimSpace(snapshot.Projects[i].Path)] = i
	}

	latestByProject := map[string]core.RunnerChatActivityInfo{}
	for _, event := range activity {
		ref := strings.TrimSpace(event.ProjectRef)
		if ref == "" {
			continue
		}
		if _, exists := latestByProject[ref]; !exists {
			latestByProject[ref] = event
		}
	}

	chatActive := make([]DashboardTaskItem, 0, len(latestByProject))
	for ref, event := range latestByProject {
		if !runnerChatEventIsActive(event, now) {
			continue
		}
		snapshot.Summary.Active++
		if idx, ok := projectIndex[ref]; ok {
			snapshot.Projects[idx].Summary.Active++
		}
		chatActive = append(chatActive, DashboardTaskItem{
			TaskID:      "chat:" + event.ID,
			Title:       chatActivityProjectName(ref),
			Provider:    "ChatGPT via Workbench",
			Status:      core.TaskRunning,
			StatusLabel: "Working",
			NextAction:  "Latest Workbench action: " + friendlyChatAction(event.Action) + ".",
		})
	}
	sort.SliceStable(chatActive, func(i, j int) bool {
		return strings.ToLower(chatActive[i].Title) < strings.ToLower(chatActive[j].Title)
	})
	combinedActive := append(chatActive, snapshot.ActiveTasks...)
	if len(combinedActive) > 6 {
		combinedActive = combinedActive[:6]
	}
	snapshot.ActiveTasks = combinedActive

	recent := make([]DashboardActivityItem, 0, len(activity)+len(snapshot.RecentActivity))
	for _, event := range activity {
		if event.UpdatedAt.Before(now.Add(-24 * time.Hour)) {
			continue
		}
		status := core.TaskCompleted
		label := "ChatGPT"
		if runnerChatEventIsActive(event, now) {
			status = core.TaskRunning
			label = "Working"
		} else if event.State == "failed" {
			status = core.TaskFailed
			label = "Failed"
		} else if event.State == "needs_attention" {
			status = core.TaskNeedsAttention
			label = "Needs you"
		} else if event.State == "waiting" || event.State == "running" {
			status = core.TaskRunning
			label = "Working"
		}
		recent = append(recent, DashboardActivityItem{
			TaskID:      "chat:" + event.ID,
			Title:       chatActivityProjectName(event.ProjectRef),
			Detail:      "ChatGPT via Workbench · " + friendlyChatAction(event.Action),
			Status:      status,
			StatusLabel: label,
			UpdatedAt:   event.UpdatedAt,
		})
		if len(recent) >= 12 {
			break
		}
	}
	recent = append(recent, snapshot.RecentActivity...)
	sort.SliceStable(recent, func(i, j int) bool { return recent[i].UpdatedAt.After(recent[j].UpdatedAt) })
	if len(recent) > 6 {
		recent = recent[:6]
	}
	snapshot.RecentActivity = recent
	return snapshot
}

func runnerChatEventIsActive(event core.RunnerChatActivityInfo, now time.Time) bool {
	if event.ActiveKnown {
		return event.Active
	}
	if strings.EqualFold(strings.TrimSpace(event.Action), "delegate_task") {
		switch strings.ToLower(strings.TrimSpace(event.State)) {
		case "running", "waiting", "needs_attention":
			return true
		default:
			return false
		}
	}
	return !event.UpdatedAt.Before(now.Add(-dashboardChatSessionWindow))
}

func chatActivityProjectName(ref string) string {
	ref = strings.TrimSpace(ref)
	ref = strings.TrimPrefix(ref, core.RunnerProjectPrefix)
	if idx := strings.LastIndex(ref, "/"); idx >= 0 && idx+1 < len(ref) {
		ref = ref[idx+1:]
	}
	if ref == "" {
		return "Cluster project"
	}
	return ref
}

func friendlyChatAction(action string) string {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "list_files":
		return "listed repository files"
	case "search_text":
		return "searched source"
	case "read_file":
		return "read source"
	case "apply_patch":
		return "applied ChatGPT's code change"
	case "run_safe_command":
		return "ran a safe build/test command"
	case "save_note":
		return "saved project context"
	case "save_memory":
		return "saved durable memory"
	case "search_memory":
		return "searched durable memory"
	case "save_context":
		return "saved continuation context"
	case "get_context":
		return "loaded continuation context"
	case "delegate_task":
		return "delegated autonomous work"
	default:
		value := strings.ReplaceAll(strings.TrimSpace(action), "_", " ")
		if value == "" {
			return "used Workbench"
		}
		return value
	}
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
