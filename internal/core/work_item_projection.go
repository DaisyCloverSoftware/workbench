package core

import (
	"fmt"
	"strings"
)

// WorkItemFromTask projects one durable Workbench task into the common
// operations-dashboard vocabulary. Task remains authoritative; this projection
// can therefore evolve without migrating historic task records.
func WorkItemFromTask(task Task, project Project) WorkItem {
	item := WorkItem{
		ID:              task.ID,
		ProjectID:       project.ID,
		ProjectName:     strings.TrimSpace(project.Name),
		Title:           strings.TrimSpace(task.Title),
		State:           workItemStateFromTask(task.Status),
		Priority:        EffectiveTaskPriority(task, project),
		Location:        taskWorkLocation(task),
		Progress:        taskWorkProgress(task),
		Blocker:         taskWorkBlocker(task),
		CreatedAt:       task.CreatedAt,
		StartedAt:       task.StartedAt,
		UpdatedAt:       task.UpdatedAt,
		CanReprioritize: taskCanReprioritize(task.Status),
		CanCancel:       taskBlocksProjectRemoval(task.Status),
		CanMove:         false,
	}
	if item.Title == "" {
		item.Title = "Workbench task"
	}
	return item
}

// WorkItemsFromTasks resolves projects by canonical path and assigns queue
// positions independently inside each execution lane.
func WorkItemsFromTasks(tasks []Task, projects []Project) []WorkItem {
	byPath := make(map[string]Project, len(projects))
	for _, project := range projects {
		byPath[normalizeProjectPath(project.Path)] = project
	}
	items := make([]WorkItem, 0, len(tasks))
	for _, task := range tasks {
		project := byPath[normalizeProjectPath(task.ProjectPath)]
		items = append(items, WorkItemFromTask(task, project))
	}
	return AssignLaneQueuePositions(items)
}

func workItemStateFromTask(status TaskStatus) WorkItemState {
	switch status {
	case TaskQueued:
		return WorkItemQueued
	case TaskRouting:
		return WorkItemRouting
	case TaskRunning:
		return WorkItemRunning
	case TaskWaitingRetry, TaskWaitingDependency:
		return WorkItemWaiting
	case TaskNeedsAttention:
		return WorkItemNeedsAttention
	case TaskCompleted:
		return WorkItemCompleted
	case TaskFailed:
		return WorkItemFailed
	case TaskCancelled:
		return WorkItemCancelled
	default:
		return WorkItemWaiting
	}
}

func taskWorkLocation(task Task) WorkLocation {
	provider := strings.TrimSpace(task.ProviderID)
	location := WorkLocation{Provider: provider}

	switch task.Status {
	case TaskNeedsAttention:
		location.Lane = WorkLaneNeedsHuman
		location.Executor = "Human decision"
		return location
	case TaskWaitingRetry:
		location.Lane = WorkLaneWaiting
		location.Executor = "Workbench scheduler"
		return location
	case TaskWaitingDependency:
		if task.Dependency != nil && task.Dependency.Kind == DependencyGitHubActions {
			location.Lane = WorkLaneCIBuilds
			location.Executor = "GitHub Actions"
			location.Tool = "GitHub Actions"
			return location
		}
		location.Lane = WorkLaneWaiting
		location.Executor = "External dependency"
		return location
	}

	if IsOperationsTask(task) {
		location.Lane = WorkLaneServerOperations
		location.Executor = provider
		if location.Executor == "" {
			location.Executor = "Operations router"
		}
		return location
	}
	location.Lane = WorkLaneAIWorkers
	location.Executor = provider
	if location.Executor == "" {
		location.Executor = "Workbench router"
	}
	return location
}

func taskWorkProgress(task Task) WorkProgress {
	switch task.Status {
	case TaskRouting:
		return WorkProgress{Kind: WorkProgressIndeterminate, StageName: "Routing"}
	case TaskRunning:
		return WorkProgress{Kind: WorkProgressIndeterminate, StageName: "Working"}
	default:
		return WorkProgress{Kind: WorkProgressNone}
	}
}

func taskWorkBlocker(task Task) string {
	switch task.Status {
	case TaskNeedsAttention:
		return strings.TrimSpace(task.AttentionQuestion)
	case TaskWaitingDependency:
		if task.Dependency == nil {
			return "Waiting on external dependency"
		}
		if reason := strings.TrimSpace(task.Dependency.Reason); reason != "" {
			return reason
		}
		if task.Dependency.Kind == DependencyGitHubActions && task.Dependency.RunID > 0 {
			return fmt.Sprintf("Waiting on GitHub Actions run %d", task.Dependency.RunID)
		}
		return "Waiting on external dependency"
	case TaskWaitingRetry:
		if task.RetryAt != nil {
			return "Scheduled retry"
		}
		return "Waiting to retry"
	default:
		return ""
	}
}

func taskCanReprioritize(status TaskStatus) bool {
	switch status {
	case TaskQueued, TaskRouting, TaskWaitingRetry, TaskWaitingDependency, TaskNeedsAttention:
		return true
	default:
		return false
	}
}
