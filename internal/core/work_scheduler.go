package core

import "sort"

// WorkLaneCapacity describes how many jobs a lane may execute concurrently.
// Capacity is deliberately explicit rather than inferred from provider count so
// Workbench can later surface and edit operator policy without lying about the
// underlying machines.
type WorkLaneCapacity struct {
	Lane     WorkLane `json:"lane"`
	Capacity int      `json:"capacity"`
}

type WorkDispatchPlan struct {
	Items      []WorkItem          `json:"items"`
	Dispatch   []WorkItem          `json:"dispatch"`
	Capacity   map[WorkLane]int    `json:"capacity"`
	Running    map[WorkLane]int    `json:"running"`
	Available  map[WorkLane]int    `json:"available"`
}

// PlanWorkDispatch assigns per-lane queue positions and identifies which queued
// items may start next given current capacity. It never pre-empts running work:
// priority changes only influence the next dispatch opportunity.
func PlanWorkDispatch(items []WorkItem, capacities []WorkLaneCapacity) WorkDispatchPlan {
	planned := AssignLaneQueuePositions(items)
	plan := WorkDispatchPlan{
		Items:     planned,
		Capacity:  map[WorkLane]int{},
		Running:   map[WorkLane]int{},
		Available: map[WorkLane]int{},
	}
	for _, entry := range capacities {
		capacity := entry.Capacity
		if capacity < 0 {
			capacity = 0
		}
		plan.Capacity[entry.Lane] = capacity
	}
	for _, item := range planned {
		if item.State == WorkItemRunning || item.State == WorkItemRouting {
			plan.Running[item.Location.Lane]++
		}
	}
	for lane, capacity := range plan.Capacity {
		available := capacity - plan.Running[lane]
		if available < 0 {
			available = 0
		}
		plan.Available[lane] = available
	}

	queuedByLane := map[WorkLane][]WorkItem{}
	for _, item := range planned {
		if item.State == WorkItemQueued {
			queuedByLane[item.Location.Lane] = append(queuedByLane[item.Location.Lane], item)
		}
	}
	lanes := make([]WorkLane, 0, len(queuedByLane))
	for lane := range queuedByLane {
		lanes = append(lanes, lane)
	}
	sort.Slice(lanes, func(i, j int) bool { return lanes[i] < lanes[j] })
	for _, lane := range lanes {
		available := plan.Available[lane]
		if available <= 0 {
			continue
		}
		queue := queuedByLane[lane]
		sort.SliceStable(queue, func(i, j int) bool { return queue[i].QueuePosition < queue[j].QueuePosition })
		if available > len(queue) {
			available = len(queue)
		}
		plan.Dispatch = append(plan.Dispatch, queue[:available]...)
	}
	return plan
}
