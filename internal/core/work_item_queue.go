package core

import "sort"

// AssignLaneQueuePositions returns a copy of items where queued work is ranked
// independently inside each execution lane. Running/waiting work keeps queue
// position zero because it is not waiting for capacity in that lane.
func AssignLaneQueuePositions(items []WorkItem) []WorkItem {
	out := append([]WorkItem(nil), items...)
	byLane := map[WorkLane][]int{}
	for i := range out {
		out[i].QueuePosition = 0
		if out[i].State != WorkItemQueued {
			continue
		}
		lane := out[i].Location.Lane
		byLane[lane] = append(byLane[lane], i)
	}

	for _, indexes := range byLane {
		sort.SliceStable(indexes, func(i, j int) bool {
			return workItemQueueLess(out[indexes[i]], out[indexes[j]])
		})
		for position, index := range indexes {
			out[index].Priority = NormalizeWorkPriority(out[index].Priority)
			out[index].QueuePosition = position + 1
		}
	}
	return out
}
