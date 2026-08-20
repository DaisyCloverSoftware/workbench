package core

import (
	"sort"
	"strings"
	"time"
)

type WorkPriority int

const (
	PriorityCritical WorkPriority = iota
	PriorityHigh
	PriorityNormal
	PriorityLow
)

func (p WorkPriority) String() string {
	switch p {
	case PriorityCritical:
		return "Critical"
	case PriorityHigh:
		return "High"
	case PriorityLow:
		return "Low"
	default:
		return "Normal"
	}
}

type WorkLane string

const (
	WorkLaneServerOps          WorkLane = "server_ops"
	WorkLaneCIBuilds           WorkLane = "ci_builds"
	WorkLaneWindowsWorkstation WorkLane = "windows_workstation"
	WorkLaneAIWorkers          WorkLane = "ai_workers"
	WorkLaneWaiting            WorkLane = "waiting"
	WorkLaneNeedsYou           WorkLane = "needs_you"
)

type ProgressKind string

const (
	ProgressIndeterminate ProgressKind = "indeterminate"
	ProgressMeasured      ProgressKind = "measured"
	ProgressStages        ProgressKind = "stages"
)

type WorkProgress struct {
	Kind       ProgressKind `json:"kind,omitempty"`
	Current    int64        `json:"current,omitempty"`
	Total      int64        `json:"total,omitempty"`
	Unit       string       `json:"unit,omitempty"`
	Phase      string       `json:"phase,omitempty"`
	Stage      int          `json:"stage,omitempty"`
	StageTotal int          `json:"stage_total,omitempty"`
}

type WorkItem struct {
	ID            string
	ProjectPath   string
	ProjectName   string
	Title         string
	State         TaskStatus
	Priority      WorkPriority
	Lane          WorkLane
	QueuePosition int
	Executor      string
	Machine       string
	Provider      string
	Progress      WorkProgress
	Dependency    string
	Commit        string
	CreatedAt     time.Time
	StartedAt     *time.Time
	UpdatedAt     time.Time
	NeedsHuman    bool
}

func DefaultTaskPriority(task Task) WorkPriority {
	if task.Priority < PriorityCritical || task.Priority > PriorityLow {
		return PriorityNormal
	}
	// Zero was historically absent from persisted tasks and is now also the
	// Critical enum value. Treat a zero priority on pre-scheduler tasks as Normal;
	// newly scheduled Critical tasks are stamped explicitly by the scheduler.
	if task.Priority == PriorityCritical && task.Progress.Kind == "" && task.Status != TaskQueued {
		return PriorityNormal
	}
	return task.Priority
}

func TaskLane(task Task) WorkLane {
	if task.Status == TaskNeedsAttention {
		return WorkLaneNeedsYou
	}
	if task.Status == TaskWaitingDependency || task.Status == TaskWaitingRetry {
		return WorkLaneWaiting
	}
	if task.Dependency != nil && task.Dependency.Kind == DependencyGitHubActions {
		return WorkLaneCIBuilds
	}
	if IsOperationsTask(task) {
		return WorkLaneServerOps
	}
	return WorkLaneAIWorkers
}

func TaskProgress(task Task) WorkProgress {
	progress := task.Progress
	if progress.Kind == ProgressMeasured && progress.Total <= 0 {
		progress.Kind = ProgressIndeterminate
	}
	if progress.Kind == ProgressStages && progress.StageTotal <= 0 {
		progress.Kind = ProgressIndeterminate
	}
	if progress.Kind == "" {
		progress.Kind = ProgressIndeterminate
	}
	if strings.TrimSpace(progress.Phase) == "" {
		switch task.Status {
		case TaskQueued:
			progress.Phase = "Queued"
		case TaskRouting:
			progress.Phase = "Selecting executor"
		case TaskRunning:
			progress.Phase = "Running"
		case TaskWaitingDependency:
			progress.Phase = "Waiting on dependency"
		case TaskWaitingRetry:
			progress.Phase = "Waiting to retry"
		case TaskNeedsAttention:
			progress.Phase = "Needs human decision"
		}
	}
	return progress
}

func QueuePositions(tasks []Task) map[string]int {
	queued := make([]Task, 0)
	for _, task := range tasks {
		if task.Status == TaskQueued {
			queued = append(queued, task)
		}
	}
	sort.SliceStable(queued, func(i, j int) bool {
		pi, pj := DefaultTaskPriority(queued[i]), DefaultTaskPriority(queued[j])
		if pi != pj {
			return pi < pj
		}
		if !queued[i].CreatedAt.Equal(queued[j].CreatedAt) {
			return queued[i].CreatedAt.Before(queued[j].CreatedAt)
		}
		return strings.Compare(queued[i].ID, queued[j].ID) < 0
	})
	positions := make(map[string]int, len(queued))
	laneCounts := map[WorkLane]int{}
	for _, task := range queued {
		lane := TaskLane(task)
		laneCounts[lane]++
		positions[task.ID] = laneCounts[lane]
	}
	return positions
}
