package desktop

import (
	"strings"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

type ProjectItem struct {
	ID         string
	Name       string
	Path       string
	Notes      string
	Pinned     bool
	Active     bool
	LastUsedAt time.Time
	Summary    core.TaskDashboardSummary
}

type TaskItem struct {
	ID                 string
	Title              string
	Intent             string
	Status              core.TaskStatus
	StatusLabel         string
	ProviderLabel       string
	NextAction          string
	RetryAt             *time.Time
	NeedsHuman          bool
	Terminal            bool
	Output              string
	Error               string
	AttentionQuestion   string
	ReviewBranch        string
	ReviewCommit        string
	ReviewFiles         int
	PublicationStatus   core.ReviewPublicationStatus
	PullRequestStatus   core.ReviewPullRequestStatus
	PullRequestNumber   int
	PullRequestState    string
}

type AttentionTarget struct {
	ProjectID string
	TaskID    string
}

type Snapshot struct {
	Projects        []ProjectItem
	ActiveProjectID string
	ActiveName      string
	ActivePath      string
	ActiveNotes     string
	Summary         core.TaskDashboardSummary
	Tasks           []TaskItem
	SelectedTaskID  string
}

func BuildSnapshot(eng *core.Engine, selectedTaskID string) Snapshot {
	if eng == nil {
		return Snapshot{}
	}
	projects := eng.Projects()
	active, hasActive := eng.ActiveProject()
	snapshot := Snapshot{Projects: make([]ProjectItem, 0, len(projects))}
	if hasActive {
		snapshot.ActiveProjectID = active.ID
		snapshot.ActiveName = active.Name
		snapshot.ActivePath = active.Path
		snapshot.ActiveNotes = active.Notes
	}

	for _, project := range projects {
		tasks := eng.TasksForProject(project.ID)
		summary := core.SummarizeTasks(tasks)
		snapshot.Projects = append(snapshot.Projects, ProjectItem{
			ID:         project.ID,
			Name:       project.Name,
			Path:       project.Path,
			Notes:      project.Notes,
			Pinned:     project.Pinned,
			Active:     hasActive && project.ID == active.ID,
			LastUsedAt: project.LastUsedAt,
			Summary:    summary,
		})
		if !hasActive || project.ID != active.ID {
			continue
		}
		snapshot.Summary = summary
		snapshot.Tasks = taskItems(tasks)
	}

	snapshot.SelectedTaskID = chooseSelectedTask(snapshot.Tasks, selectedTaskID)
	return snapshot
}

// FirstAttentionTarget returns the first genuine human-attention task in the
// same pinned/recent project order used by the production sidebar. It is
// deliberately read-only; the Windows action performs the explicit project
// selection only after the human clicks Needs you.
func FirstAttentionTarget(eng *core.Engine) (AttentionTarget, bool) {
	if eng == nil {
		return AttentionTarget{}, false
	}
	for _, project := range eng.Projects() {
		for _, task := range eng.TasksForProject(project.ID) {
			if task.Status == core.TaskNeedsAttention {
				return AttentionTarget{ProjectID: project.ID, TaskID: task.ID}, true
			}
		}
	}
	return AttentionTarget{}, false
}

func taskItems(tasks []core.Task) []TaskItem {
	items := make([]TaskItem, 0, len(tasks))
	for _, task := range tasks {
		presentation := core.PresentTask(task)
		nextAction := presentation.NextAction
		if presentation.RetryAt != nil {
			nextAction += " Next attempt: " + presentation.RetryAt.Local().Format("2 Jan 15:04:05") + "."
		}
		item := TaskItem{
			ID:                 task.ID,
			Title:              task.Title,
			Intent:             task.Intent,
			Status:             task.Status,
			StatusLabel:        presentation.StatusLabel,
			ProviderLabel:      presentation.ProviderLabel,
			NextAction:         nextAction,
			RetryAt:            presentation.RetryAt,
			NeedsHuman:         presentation.NeedsHuman,
			Terminal:           presentation.Terminal,
			Output:             task.Output,
			Error:              task.Error,
			AttentionQuestion:  task.AttentionQuestion,
			ReviewBranch:       presentation.ReviewBranch,
			ReviewCommit:       presentation.ReviewCommit,
			ReviewFiles:        presentation.ReviewFiles,
			PublicationStatus:  presentation.PublicationStatus,
		}
		if task.Review != nil {
			item.PullRequestStatus = task.Review.PullRequestStatus
			item.PullRequestNumber = task.Review.PullRequestNumber
			item.PullRequestState = task.Review.PullRequestState
		}
		items = append(items, item)
	}
	return items
}

func chooseSelectedTask(tasks []TaskItem, requested string) string {
	requested = strings.TrimSpace(requested)
	if requested != "" {
		for _, task := range tasks {
			if task.ID == requested {
				return task.ID
			}
		}
	}
	for _, task := range tasks {
		if task.NeedsHuman {
			return task.ID
		}
	}
	for _, task := range tasks {
		switch task.Status {
		case core.TaskQueued, core.TaskRouting, core.TaskRunning, core.TaskWaitingRetry:
			return task.ID
		}
	}
	if len(tasks) > 0 {
		return tasks[0].ID
	}
	return ""
}

func (s Snapshot) SelectedTask() (TaskItem, bool) {
	for _, task := range s.Tasks {
		if task.ID == s.SelectedTaskID {
			return task, true
		}
	}
	return TaskItem{}, false
}
