package core

import (
	"sort"
	"strings"
	"time"
)

// WorkPriority is the user-visible scheduling priority shared by Workbench
// execution lanes. Higher priority work sorts ahead of lower priority work,
// while FIFO ordering is retained within the same priority.
type WorkPriority string

const (
	WorkPriorityCritical WorkPriority = "critical"
	WorkPriorityHigh     WorkPriority = "high"
	WorkPriorityNormal   WorkPriority = "normal"
	WorkPriorityLow      WorkPriority = "low"
)

// WorkLane identifies the execution capacity a work item consumes. It is a
// scheduling/resource concept rather than a provider name: multiple providers
// or machines can supply capacity to the same lane.
type WorkLane string

const (
	WorkLaneServerOperations WorkLane = "server_operations"
	WorkLaneCIBuilds         WorkLane = "ci_builds"
	WorkLaneWindowsHost      WorkLane = "windows_host"
	WorkLaneAIWorkers        WorkLane = "ai_workers"
	WorkLaneWaiting          WorkLane = "waiting"
	WorkLaneNeedsHuman       WorkLane = "needs_human"
)

// WorkItemState is the cross-subsystem state vocabulary used by the operations
// dashboard. Source-specific adapters remain responsible for mapping their
// durable native states into these values.
type WorkItemState string

const (
	WorkItemQueued         WorkItemState = "queued"
	WorkItemRouting        WorkItemState = "routing"
	WorkItemRunning        WorkItemState = "running"
	WorkItemWaiting        WorkItemState = "waiting"
	WorkItemNeedsAttention WorkItemState = "needs_attention"
	WorkItemCompleted      WorkItemState = "completed"
	WorkItemFailed         WorkItemState = "failed"
	WorkItemCancelled      WorkItemState = "cancelled"
)

// WorkProgressKind distinguishes measured progress from phase progress and
// genuinely indeterminate work. The dashboard must not fabricate a percentage
// for indeterminate work.
type WorkProgressKind string

const (
	WorkProgressNone          WorkProgressKind = "none"
	WorkProgressMeasured      WorkProgressKind = "measured"
	WorkProgressStages        WorkProgressKind = "stages"
	WorkProgressIndeterminate WorkProgressKind = "indeterminate"
)

type WorkProgress struct {
	Kind       WorkProgressKind `json:"kind"`
	Current    int64            `json:"current,omitempty"`
	Total      int64            `json:"total,omitempty"`
	Unit       string           `json:"unit,omitempty"`
	Stage      int              `json:"stage,omitempty"`
	StageTotal int              `json:"stage_total,omitempty"`
	StageName  string           `json:"stage_name,omitempty"`
}

// Percent reports a bounded integer percentage only when Workbench has real
// measured or stage-count progress. Indeterminate and absent progress return
// ok=false so callers render an activity bar/timer instead of a fake number.
func (p WorkProgress) Percent() (percent int, ok bool) {
	switch p.Kind {
	case WorkProgressMeasured:
		if p.Total <= 0 || p.Current < 0 {
			return 0, false
		}
		current := p.Current
		if current > p.Total {
			current = p.Total
		}
		return int((current * 100) / p.Total), true
	case WorkProgressStages:
		if p.StageTotal <= 0 || p.Stage < 0 {
			return 0, false
		}
		stage := p.Stage
		if stage > p.StageTotal {
			stage = p.StageTotal
		}
		return (stage * 100) / p.StageTotal, true
	default:
		return 0, false
	}
}

// WorkLocation describes where an item is executing without conflating the
// scheduling lane with a particular host, runner or tool.
type WorkLocation struct {
	Lane     WorkLane `json:"lane"`
	Executor string   `json:"executor,omitempty"`
	Machine  string   `json:"machine,omitempty"`
	Provider string   `json:"provider,omitempty"`
	Tool     string   `json:"tool,omitempty"`
}

// WorkItem is the common dashboard projection for durable Workbench tasks,
// operations controls, CI dependencies and host-bridge jobs. It is deliberately
// descriptive: subsystem-specific durable records remain authoritative.
type WorkItem struct {
	ID             string       `json:"id"`
	ProjectID      string       `json:"project_id,omitempty"`
	ProjectName    string       `json:"project_name,omitempty"`
	Title          string       `json:"title"`
	State          WorkItemState `json:"state"`
	Priority       WorkPriority `json:"priority"`
	QueuePosition  int          `json:"queue_position,omitempty"`
	Location       WorkLocation `json:"location"`
	Progress       WorkProgress `json:"progress"`
	Blocker        string       `json:"blocker,omitempty"`
	Commit         string       `json:"commit,omitempty"`
	CreatedAt      time.Time    `json:"created_at"`
	StartedAt      *time.Time   `json:"started_at,omitempty"`
	UpdatedAt      time.Time    `json:"updated_at"`
	CanReprioritize bool        `json:"can_reprioritize,omitempty"`
	CanCancel       bool        `json:"can_cancel,omitempty"`
	CanMove         bool        `json:"can_move,omitempty"`
}

func NormalizeWorkPriority(priority WorkPriority) WorkPriority {
	switch WorkPriority(strings.ToLower(strings.TrimSpace(string(priority)))) {
	case WorkPriorityCritical:
		return WorkPriorityCritical
	case WorkPriorityHigh:
		return WorkPriorityHigh
	case WorkPriorityLow:
		return WorkPriorityLow
	default:
		return WorkPriorityNormal
	}
}

func WorkPriorityRank(priority WorkPriority) int {
	switch NormalizeWorkPriority(priority) {
	case WorkPriorityCritical:
		return 0
	case WorkPriorityHigh:
		return 1
	case WorkPriorityNormal:
		return 2
	case WorkPriorityLow:
		return 3
	default:
		return 2
	}
}

// OrderQueuedWorkItems returns queued items in scheduler order and assigns
// one-based queue positions. Priority wins first; creation time preserves FIFO;
// the stable item ID gives deterministic ordering for identical timestamps.
func OrderQueuedWorkItems(items []WorkItem) []WorkItem {
	queued := make([]WorkItem, 0, len(items))
	for _, item := range items {
		if item.State != WorkItemQueued {
			continue
		}
		copy := item
		copy.Priority = NormalizeWorkPriority(copy.Priority)
		copy.QueuePosition = 0
		queued = append(queued, copy)
	}
	sort.SliceStable(queued, func(i, j int) bool {
		leftRank := WorkPriorityRank(queued[i].Priority)
		rightRank := WorkPriorityRank(queued[j].Priority)
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		if !queued[i].CreatedAt.Equal(queued[j].CreatedAt) {
			return queued[i].CreatedAt.Before(queued[j].CreatedAt)
		}
		return strings.ToLower(queued[i].ID) < strings.ToLower(queued[j].ID)
	})
	for i := range queued {
		queued[i].QueuePosition = i + 1
	}
	return queued
}
