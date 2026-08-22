package desktop

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

const dashboardRecentOutcomeLimit = 8

type DashboardOperationsProjectActivity struct {
	Key        string
	Name       string
	Path       string
	Total      int
	Queued     int
	Running    int
	Waiting    int
	NeedsHuman int
}

type DashboardOperationsWorkerAssignment struct {
	Worker  string
	Running int
	TaskIDs []string
}

type DashboardOperationsDetail struct {
	Item             core.WorkItem
	Project          string
	Task             string
	Worker           string
	State            string
	Progress         string
	LatestActivity   string
	WaitReason       string
	WaitSince        time.Time
	AutoContinuation string
	Failure          string
	Reference        string
	OwnerAction      string
	LocalTask        bool
	CanReprioritize  bool
}

type DashboardOperationsSurface struct {
	Live           DashboardOperationsSnapshot
	Details        map[string]DashboardOperationsDetail
	Projects       []DashboardOperationsProjectActivity
	Workers        []DashboardOperationsWorkerAssignment
	RecentOutcomes []core.WorkItem
}

func BuildDashboardOperationsSurface(eng *core.Engine) DashboardOperationsSurface {
	if eng != nil {
		ensureRunnerChatActivityMonitor(eng)
	}
	return buildDashboardOperationsSurface(eng, runnerChatActivitySnapshot())
}

func buildDashboardOperationsSurface(eng *core.Engine, remote []core.RunnerChatActivityInfo) DashboardOperationsSurface {
	live := buildDashboardOperationsSnapshot(eng, remote)
	out := DashboardOperationsSurface{
		Live:    live,
		Details: map[string]DashboardOperationsDetail{},
	}
	if eng == nil {
		return out
	}

	st := eng.State()
	projectNames := map[string]string{}
	for _, project := range eng.Projects() {
		projectNames[strings.TrimSpace(project.Path)] = strings.TrimSpace(project.Name)
	}
	tasksByID := map[string]core.Task{}
	for _, task := range st.Tasks {
		tasksByID[task.ID] = task
	}
	remoteByID := map[string]core.RunnerChatActivityInfo{}
	for _, event := range remote {
		remoteByID[event.ID] = event
	}

	projects := map[string]*DashboardOperationsProjectActivity{}
	workers := map[string]*DashboardOperationsWorkerAssignment{}
	for _, item := range live.Items {
		if task, ok := tasksByID[item.ID]; ok {
			out.Details[item.ID] = operationsDetailForTask(item, task)
		} else if event, ok := remoteByID[item.ID]; ok {
			out.Details[item.ID] = operationsDetailForRemote(item, event)
		} else {
			out.Details[item.ID] = operationsDetailForItem(item)
		}
		addOperationsProjectActivity(projects, item)
		if item.State == core.TaskRunning || item.State == core.TaskRouting {
			addOperationsWorkerAssignment(workers, item)
		}
	}

	seenOutcome := map[string]bool{}
	for _, task := range st.Tasks {
		if task.Archived || (task.Status != core.TaskCompleted && task.Status != core.TaskFailed) {
			continue
		}
		item := operationsWorkItemForTask(task, projectNames)
		out.RecentOutcomes = append(out.RecentOutcomes, item)
		out.Details[item.ID] = operationsDetailForTask(item, task)
		seenOutcome[item.ID] = true
	}
	for _, event := range remote {
		if seenOutcome[event.ID] {
			continue
		}
		status, ok := remoteActivityTerminalStatus(event.State)
		if !ok {
			continue
		}
		item := operationsWorkItemForRemoteOutcome(event, status)
		out.RecentOutcomes = append(out.RecentOutcomes, item)
		out.Details[item.ID] = operationsDetailForRemote(item, event)
		seenOutcome[item.ID] = true
	}
	sort.SliceStable(out.RecentOutcomes, func(i, j int) bool {
		if !out.RecentOutcomes[i].UpdatedAt.Equal(out.RecentOutcomes[j].UpdatedAt) {
			return out.RecentOutcomes[i].UpdatedAt.After(out.RecentOutcomes[j].UpdatedAt)
		}
		return out.RecentOutcomes[i].ID < out.RecentOutcomes[j].ID
	})
	if len(out.RecentOutcomes) > dashboardRecentOutcomeLimit {
		out.RecentOutcomes = out.RecentOutcomes[:dashboardRecentOutcomeLimit]
	}

	for _, project := range projects {
		out.Projects = append(out.Projects, *project)
	}
	sort.SliceStable(out.Projects, func(i, j int) bool {
		if out.Projects[i].Total != out.Projects[j].Total {
			return out.Projects[i].Total > out.Projects[j].Total
		}
		return strings.ToLower(out.Projects[i].Name) < strings.ToLower(out.Projects[j].Name)
	})
	for _, worker := range workers {
		worker.TaskIDs = append([]string(nil), worker.TaskIDs...)
		out.Workers = append(out.Workers, *worker)
	}
	sort.SliceStable(out.Workers, func(i, j int) bool {
		if out.Workers[i].Running != out.Workers[j].Running {
			return out.Workers[i].Running > out.Workers[j].Running
		}
		return strings.ToLower(out.Workers[i].Worker) < strings.ToLower(out.Workers[j].Worker)
	})
	return out
}

func addOperationsProjectActivity(projects map[string]*DashboardOperationsProjectActivity, item core.WorkItem) {
	key := strings.TrimSpace(item.ProjectPath)
	name := strings.TrimSpace(item.ProjectName)
	if key == "" {
		key = "workbench://local"
	}
	if name == "" {
		name = chatActivityProjectName(item.ProjectPath)
	}
	if name == "" {
		name = "Workbench"
	}
	project := projects[key]
	if project == nil {
		project = &DashboardOperationsProjectActivity{Key: key, Name: name, Path: strings.TrimSpace(item.ProjectPath)}
		projects[key] = project
	}
	project.Total++
	switch item.State {
	case core.TaskQueued:
		project.Queued++
	case core.TaskRouting, core.TaskRunning:
		project.Running++
	case core.TaskWaitingDependency, core.TaskWaitingRetry:
		project.Waiting++
	case core.TaskNeedsAttention:
		project.NeedsHuman++
	}
}

func addOperationsWorkerAssignment(workers map[string]*DashboardOperationsWorkerAssignment, item core.WorkItem) {
	workerName := strings.TrimSpace(item.Provider)
	if workerName == "" {
		workerName = strings.TrimSpace(item.Executor)
	}
	if workerName == "" {
		workerName = "Routing"
	}
	key := strings.ToLower(workerName)
	worker := workers[key]
	if worker == nil {
		worker = &DashboardOperationsWorkerAssignment{Worker: workerName}
		workers[key] = worker
	}
	worker.Running++
	worker.TaskIDs = append(worker.TaskIDs, item.ID)
}

func operationsWorkItemForTask(task core.Task, projectNames map[string]string) core.WorkItem {
	name := projectNames[strings.TrimSpace(task.ProjectPath)]
	if name == "" {
		name = chatActivityProjectName(task.ProjectPath)
	}
	provider := strings.TrimSpace(task.ProviderID)
	if provider == "" {
		provider = "Workbench"
	}
	item := core.WorkItem{
		ID:          task.ID,
		ProjectPath: task.ProjectPath,
		ProjectName: name,
		Title:       task.Title,
		State:       task.Status,
		Priority:    core.DefaultTaskPriority(task),
		Lane:        core.TaskLane(task),
		Provider:    provider,
		Progress:    core.TaskProgress(task),
		CreatedAt:   task.CreatedAt,
		StartedAt:   task.StartedAt,
		UpdatedAt:   task.UpdatedAt,
		NeedsHuman:  task.Status == core.TaskNeedsAttention,
	}
	if task.Dependency != nil {
		item.Dependency = dependencySummary(task)
	}
	if task.Review != nil {
		item.Commit = strings.TrimSpace(task.Review.Commit)
	}
	return item
}

func operationsWorkItemForRemoteOutcome(event core.RunnerChatActivityInfo, status core.TaskStatus) core.WorkItem {
	projectName := chatActivityProjectName(event.ProjectRef)
	if projectName == "" {
		projectName = "Workbench"
	}
	return core.WorkItem{
		ID:          event.ID,
		ProjectPath: event.ProjectRef,
		ProjectName: projectName,
		Title:       remoteActivityTitle(event.Action),
		State:       status,
		Priority:    core.PriorityNormal,
		Lane:        remoteActivityLane(event.Action, status),
		Provider:    remoteActivityProvider(event.Action),
		Progress: core.WorkProgress{
			Kind:  core.ProgressStages,
			Phase: dashboardStatusLabel(status, string(status)),
			Stage: 1, StageTotal: 1,
		},
		UpdatedAt: event.UpdatedAt,
	}
}

func remoteActivityTerminalStatus(state string) (core.TaskStatus, bool) {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "completed", "success", "succeeded":
		return core.TaskCompleted, true
	case "failed", "error":
		return core.TaskFailed, true
	default:
		return "", false
	}
}

func operationsDetailForTask(item core.WorkItem, task core.Task) DashboardOperationsDetail {
	presentation := core.PresentTask(task)
	detail := DashboardOperationsDetail{
		Item:            item,
		Project:         operationsProjectLabel(item),
		Task:            strings.TrimSpace(task.Title),
		Worker:          strings.TrimSpace(task.ProviderID),
		State:           presentation.StatusLabel,
		Progress:        operationsProgressSummary(core.TaskProgress(task), task.Status),
		LatestActivity:  operationsLatestTaskActivity(task),
		Reference:       operationsTaskReference(task),
		OwnerAction:     strings.TrimSpace(presentation.NextAction),
		LocalTask:       true,
		CanReprioritize: task.Status == core.TaskQueued,
	}
	if detail.Worker == "" {
		detail.Worker = strings.TrimSpace(item.Provider)
	}
	if detail.Worker == "" {
		detail.Worker = presentation.ProviderLabel
	}
	if task.Status == core.TaskNeedsAttention {
		question := operationsCompactText(task.AttentionQuestion, 500)
		if question != "" {
			detail.OwnerAction = question + " " + detail.OwnerAction
		}
	}
	if task.Status == core.TaskWaitingDependency {
		detail.WaitReason = operationsDependencyWaitReason(task)
		if task.Dependency != nil && !task.Dependency.StartedAt.IsZero() {
			detail.WaitSince = task.Dependency.StartedAt
		} else {
			detail.WaitSince = task.UpdatedAt
		}
		if task.Dependency != nil && !task.Dependency.NextCheckAt.IsZero() {
			detail.AutoContinuation = "Yes — Workbench will check again at " + task.Dependency.NextCheckAt.Local().Format("2 Jan 15:04:05") + "."
		} else {
			detail.AutoContinuation = "Yes — Workbench will keep checking the dependency automatically."
		}
	}
	if task.Status == core.TaskWaitingRetry {
		detail.WaitReason = "Waiting for the next automatic worker retry after a temporary provider or capacity condition."
		if latest := operationsLastAttempt(task); latest != "" {
			detail.WaitReason += " Latest worker result: " + latest
		}
		detail.WaitSince = task.UpdatedAt
		if task.RetryAt != nil {
			detail.AutoContinuation = "Yes — Workbench will retry automatically at " + task.RetryAt.Local().Format("2 Jan 15:04:05") + "."
		} else {
			detail.AutoContinuation = "Yes — Workbench will retry automatically when the scheduler permits."
		}
	}
	if task.Status == core.TaskFailed {
		detail.Failure = operationsCompactText(task.Error, 1200)
		if detail.Failure == "" {
			detail.Failure = operationsLastAttempt(task)
		}
	}
	return detail
}

func operationsDetailForRemote(item core.WorkItem, event core.RunnerChatActivityInfo) DashboardOperationsDetail {
	detail := operationsDetailForItem(item)
	detail.LatestActivity = "Runner activity: " + remoteActivityTitle(event.Action) + "."
	detail.Reference = "Workbench runner operation " + strings.TrimSpace(event.ID)
	if item.State == core.TaskWaitingDependency || item.State == core.TaskWaitingRetry {
		detail.WaitReason = "Waiting for the runner-side dependency or retry condition reported by Workbench."
		detail.WaitSince = event.UpdatedAt
		detail.AutoContinuation = "Yes — Workbench refreshes runner state automatically and the runner continues its own dependency/retry checks."
	}
	if item.State == core.TaskNeedsAttention {
		detail.OwnerAction = "Open the coordinating Workbench/ChatGPT task and provide the requested owner decision."
	}
	if item.State == core.TaskFailed {
		detail.Failure = "Remote Workbench execution reported failure. The coordinating task retains the runner-side failure summary."
	}
	return detail
}

func operationsDetailForItem(item core.WorkItem) DashboardOperationsDetail {
	return DashboardOperationsDetail{
		Item:        item,
		Project:     operationsProjectLabel(item),
		Task:        strings.TrimSpace(item.Title),
		Worker:      strings.TrimSpace(item.Provider),
		State:       dashboardStatusLabel(item.State, string(item.State)),
		Progress:    operationsProgressSummary(item.Progress, item.State),
		Reference:   strings.TrimSpace(item.Commit),
		OwnerAction: "Workbench will continue automatically unless this item enters Needs You.",
	}
}

func operationsProjectLabel(item core.WorkItem) string {
	name := strings.TrimSpace(item.ProjectName)
	if name != "" {
		return name
	}
	name = chatActivityProjectName(item.ProjectPath)
	if name != "" {
		return name
	}
	return "Workbench"
}

func operationsProgressSummary(progress core.WorkProgress, status core.TaskStatus) string {
	phase := strings.TrimSpace(progress.Phase)
	if phase == "" {
		phase = dashboardStatusLabel(status, string(status))
	}
	switch progress.Kind {
	case core.ProgressMeasured:
		if progress.Total > 0 {
			current := progress.Current
			if current < 0 {
				current = 0
			}
			if current > progress.Total {
				current = progress.Total
			}
			unit := strings.TrimSpace(progress.Unit)
			if unit != "" {
				unit = " " + unit
			}
			percent := (current * 100) / progress.Total
			return fmt.Sprintf("%s · %d/%d%s · %d%%", phase, current, progress.Total, unit, percent)
		}
	case core.ProgressStages:
		if progress.StageTotal > 0 {
			stage := progress.Stage
			if stage < 0 {
				stage = 0
			}
			return fmt.Sprintf("%s · stage %d/%d", phase, stage, progress.StageTotal)
		}
	}
	if status == core.TaskRunning || status == core.TaskRouting {
		return phase + " · deterministic percentage unavailable"
	}
	return phase
}

func operationsLatestTaskActivity(task core.Task) string {
	if latest := operationsLastAttempt(task); latest != "" {
		return latest
	}
	if value := operationsCompactText(task.RouteReason, 500); value != "" {
		return value
	}
	if value := operationsCompactText(task.DependencyResult, 500); value != "" {
		return value
	}
	if task.Status == core.TaskFailed {
		if value := operationsCompactText(task.Error, 500); value != "" {
			return value
		}
	}
	if task.Status == core.TaskCompleted {
		if value := operationsCompactText(task.Output, 500); value != "" {
			return value
		}
	}
	return strings.TrimSpace(core.TaskProgress(task).Phase)
}

func operationsLastAttempt(task core.Task) string {
	for i := len(task.Attempts) - 1; i >= 0; i-- {
		if value := operationsCompactText(task.Attempts[i], 700); value != "" {
			return value
		}
	}
	return ""
}

func operationsDependencyWaitReason(task core.Task) string {
	if task.Dependency == nil {
		return "Waiting for an external dependency."
	}
	parts := []string{dependencySummary(task)}
	if reason := operationsCompactText(task.Dependency.Reason, 500); reason != "" && !strings.EqualFold(reason, parts[0]) {
		parts = append(parts, reason)
	}
	if state := strings.TrimSpace(task.Dependency.State); state != "" {
		parts = append(parts, "state: "+state)
	}
	return strings.Join(parts, " · ")
}

func operationsTaskReference(task core.Task) string {
	if task.Dependency != nil && task.Dependency.Kind == core.DependencyGitHubActions {
		if task.Dependency.RunID > 0 {
			if repo := strings.TrimSpace(task.Dependency.Repository); repo != "" {
				return fmt.Sprintf("GitHub Actions %s run %d", repo, task.Dependency.RunID)
			}
			return fmt.Sprintf("GitHub Actions run %d", task.Dependency.RunID)
		}
		return "GitHub Actions"
	}
	if task.Review != nil && task.Review.Changed {
		if task.Review.PullRequestStatus == core.ReviewPullRequestAvailable && task.Review.PullRequestNumber > 0 {
			state := strings.TrimSpace(task.Review.PullRequestState)
			if state == "" {
				state = "available"
			}
			return fmt.Sprintf("GitHub PR #%d (%s)", task.Review.PullRequestNumber, state)
		}
		branch := strings.TrimSpace(task.Review.Branch)
		commit := strings.TrimSpace(task.Review.Commit)
		if branch != "" && commit != "" {
			return branch + " @ " + shortOperationsCommit(commit)
		}
		if commit != "" {
			return shortOperationsCommit(commit)
		}
	}
	return ""
}

func shortOperationsCommit(commit string) string {
	commit = strings.TrimSpace(commit)
	if len(commit) > 10 {
		return commit[:10]
	}
	return commit
}

func operationsCompactText(value string, limit int) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if value == "" {
		return ""
	}
	if core.LooksSecret(value) {
		return "Activity detail withheld because it looks secret."
	}
	runes := []rune(value)
	if limit > 0 && len(runes) > limit {
		return string(runes[:limit]) + "…"
	}
	return value
}

func higherOperationsPriority(priority core.WorkPriority) (core.WorkPriority, bool) {
	switch priority {
	case core.PriorityLow:
		return core.PriorityNormal, true
	case core.PriorityNormal:
		return core.PriorityHigh, true
	case core.PriorityHigh:
		return core.PriorityCritical, true
	default:
		return priority, false
	}
}

func lowerOperationsPriority(priority core.WorkPriority) (core.WorkPriority, bool) {
	switch priority {
	case core.PriorityCritical:
		return core.PriorityHigh, true
	case core.PriorityHigh:
		return core.PriorityNormal, true
	case core.PriorityNormal:
		return core.PriorityLow, true
	default:
		return priority, false
	}
}

func dashboardOperationsInvariantError(surface DashboardOperationsSurface) error {
	var queued, running, waiting, needsHuman int
	itemIDs := map[string]core.WorkItem{}
	for _, item := range surface.Live.Items {
		if item.State == core.TaskCompleted || item.State == core.TaskFailed || item.State == core.TaskCancelled {
			return fmt.Errorf("terminal item %s is present in the live operations set", item.ID)
		}
		if _, exists := itemIDs[item.ID]; exists {
			return fmt.Errorf("live item %s is duplicated", item.ID)
		}
		itemIDs[item.ID] = item
		switch item.State {
		case core.TaskQueued:
			queued++
		case core.TaskRouting, core.TaskRunning:
			running++
		case core.TaskWaitingDependency, core.TaskWaitingRetry:
			waiting++
		case core.TaskNeedsAttention:
			needsHuman++
		}
		if _, ok := surface.Details[item.ID]; !ok {
			return fmt.Errorf("live item %s has no drill-down detail", item.ID)
		}
	}
	if queued != surface.Live.Queued || running != surface.Live.Running || waiting != surface.Live.Waiting || needsHuman != surface.Live.NeedsHuman {
		return fmt.Errorf("live totals diverge from canonical items: got q=%d r=%d w=%d n=%d, snapshot q=%d r=%d w=%d n=%d", queued, running, waiting, needsHuman, surface.Live.Queued, surface.Live.Running, surface.Live.Waiting, surface.Live.NeedsHuman)
	}

	laneItems := 0
	laneSeen := map[string]bool{}
	for lane, items := range surface.Live.ByLane {
		laneItems += len(items)
		for _, item := range items {
			if item.Lane != lane {
				return fmt.Errorf("item %s is in lane %s but declares %s", item.ID, lane, item.Lane)
			}
			if laneSeen[item.ID] {
				return fmt.Errorf("item %s appears in more than one lane", item.ID)
			}
			laneSeen[item.ID] = true
		}
	}
	if laneItems != len(surface.Live.Items) {
		return fmt.Errorf("lane contents=%d but live items=%d", laneItems, len(surface.Live.Items))
	}

	projectItems := 0
	for _, project := range surface.Projects {
		if project.Total != project.Queued+project.Running+project.Waiting+project.NeedsHuman {
			return fmt.Errorf("project %s total=%d but state counts sum to %d", project.Name, project.Total, project.Queued+project.Running+project.Waiting+project.NeedsHuman)
		}
		projectItems += project.Total
	}
	if projectItems != len(surface.Live.Items) {
		return fmt.Errorf("project activity=%d but live items=%d", projectItems, len(surface.Live.Items))
	}

	workerItems := 0
	for _, worker := range surface.Workers {
		if worker.Running != len(worker.TaskIDs) {
			return fmt.Errorf("worker %s count=%d but assignments=%d", worker.Worker, worker.Running, len(worker.TaskIDs))
		}
		workerItems += worker.Running
	}
	if workerItems != surface.Live.Running {
		return fmt.Errorf("worker assignments=%d but running total=%d", workerItems, surface.Live.Running)
	}

	for _, item := range surface.RecentOutcomes {
		if item.State != core.TaskCompleted && item.State != core.TaskFailed {
			return fmt.Errorf("recent outcome %s has non-terminal state %s", item.ID, item.State)
		}
		if _, ok := surface.Details[item.ID]; !ok {
			return fmt.Errorf("recent outcome %s has no drill-down detail", item.ID)
		}
	}
	return nil
}
