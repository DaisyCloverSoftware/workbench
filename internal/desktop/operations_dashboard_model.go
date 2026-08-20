package desktop

import (
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

// OperationsDashboardSnapshot is the control-room view of work that is active
// now. It deliberately sits alongside the legacy dashboard snapshot while the
// UI migrates, so the scheduling model can become trustworthy before old panels
// are removed.
type OperationsDashboardSnapshot struct {
	GeneratedAt time.Time
	Items       []core.WorkItem
	Lanes       []DashboardWorkLane
	Running     int
	Queued      int
	Waiting     int
	NeedsHuman  int
}

type DashboardWorkLane struct {
	Lane           core.WorkLane
	Label          string
	Items          []core.WorkItem
	Running        int
	Queued         int
	Waiting        int
	NeedsAttention int
}

func BuildOperationsDashboardSnapshot(eng *core.Engine) OperationsDashboardSnapshot {
	now := time.Now().UTC()
	if eng == nil {
		return OperationsDashboardSnapshot{GeneratedAt: now, Lanes: emptyDashboardWorkLanes()}
	}

	state := eng.State()
	activeTasks := make([]core.Task, 0, len(state.Tasks))
	for _, task := range state.Tasks {
		if task.Archived || !operationsDashboardTaskIsActive(task.Status) {
			continue
		}
		activeTasks = append(activeTasks, task)
	}
	items := core.WorkItemsFromTasks(activeTasks, eng.Projects())

	// Windows host jobs are already durable Workbench-owned records, so they can
	// join the dashboard without polling the cluster or inventing another queue.
	// Inventory failures are non-fatal to the desktop: the lane remains visible
	// and the rest of the control room still renders.
	if jobs, err := core.ListHostBridgeJobs(); err == nil {
		hosts, _ := core.ListHostBridgeHosts()
		items = append(items, core.ActiveWorkItemsFromHostJobs(jobs, hosts)...)
	}
	items = core.AssignLaneQueuePositions(items)

	result := summarizeOperationsDashboard(items)
	result.GeneratedAt = now
	return result
}

func summarizeOperationsDashboard(items []core.WorkItem) OperationsDashboardSnapshot {
	result := OperationsDashboardSnapshot{
		Items: append([]core.WorkItem(nil), items...),
		Lanes: emptyDashboardWorkLanes(),
	}
	index := make(map[core.WorkLane]int, len(result.Lanes))
	for i := range result.Lanes {
		index[result.Lanes[i].Lane] = i
	}

	for _, item := range items {
		laneIndex, ok := index[item.Location.Lane]
		if !ok {
			laneIndex = index[core.WorkLaneWaiting]
		}
		lane := &result.Lanes[laneIndex]
		lane.Items = append(lane.Items, item)
		switch item.State {
		case core.WorkItemQueued:
			lane.Queued++
			result.Queued++
		case core.WorkItemRouting, core.WorkItemRunning:
			lane.Running++
			result.Running++
		case core.WorkItemWaiting:
			lane.Waiting++
			result.Waiting++
		case core.WorkItemNeedsAttention:
			lane.NeedsAttention++
			result.NeedsHuman++
		}
	}
	return result
}

func emptyDashboardWorkLanes() []DashboardWorkLane {
	return []DashboardWorkLane{
		{Lane: core.WorkLaneServerOperations, Label: "Server operations"},
		{Lane: core.WorkLaneCIBuilds, Label: "CI / builds"},
		{Lane: core.WorkLaneWindowsHost, Label: "Windows workstation"},
		{Lane: core.WorkLaneAIWorkers, Label: "AI workers"},
		{Lane: core.WorkLaneWaiting, Label: "Waiting"},
		{Lane: core.WorkLaneNeedsHuman, Label: "Needs you"},
	}
}

func operationsDashboardTaskIsActive(status core.TaskStatus) bool {
	switch status {
	case core.TaskQueued, core.TaskRouting, core.TaskRunning, core.TaskWaitingRetry, core.TaskWaitingDependency, core.TaskNeedsAttention:
		return true
	default:
		return false
	}
}
