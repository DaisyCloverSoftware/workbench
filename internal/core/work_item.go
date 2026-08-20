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
	Kind       ProgressKind
	Current    int64
	Total      int64
	Unit       string
	Phase      string
	Stage      int
	StageTotal int
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

// Until task priority is persisted by the scheduler tranche, existing tasks are
// truthfully Normal rather than being assigned a cosmetic priority in the UI.
func DefaultTaskPriority(task Task) WorkPriority { return PriorityNormal }

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
	phase := ""
	switch task.Status {
	case TaskQueued:
		phase = "Queued"
	case TaskRouting:
		phase = "Selecting executor"
	case TaskRunning:
		phase = "Running"
	case TaskWaitingDependency:
		phase = "Waiting on dependency"
	case TaskWaitingRetry:
		phase = "Waiting to retry"
	case TaskNeedsAttention:
		phase = "Needs human decision"
	}
	return WorkProgress{Kind: ProgressIndeterminate, Phase: phase}
}

func QueuePositions(tasks []Task) map[string]int {
	queued := make([]Task, 0)
	for _, task := range tasks {
		if task.Status == TaskQueued {
			queued = append(queued, task)
		}
	}
	sort.SliceStable(queued, func(i, j int) bool {
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
