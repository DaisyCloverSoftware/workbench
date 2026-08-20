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
			left := out[indexes[i]]
			right := out[indexes[j]]
			leftRank := WorkPriorityRank(left.Priority)
			rightRank := WorkPriorityRank(right.Priority)
			if leftRank != rightRank {
				return leftRank < rightRank
			}
			if !left.CreatedAt.Equal(right.CreatedAt) {
				return left.CreatedAt.Before(right.CreatedAt)
			}
			return left.ID < right.ID
		})
		for position, index := range indexes {
			out[index].Priority = NormalizeWorkPriority(out[index].Priority)
			out[index].QueuePosition = position + 1
		}
	}
	return out
}
