package core

import "strings"

const RelayOperationsIntentPrefix = "[workbench:operations]"

func hasRelayOperationsMarker(intent string) bool {
	intent = strings.ToLower(strings.TrimSpace(intent))
	idx := strings.Index(intent, RelayOperationsIntentPrefix)
	return idx >= 0 && idx < 160
}

// TaskUsesOperationsLane is a semantic alias used at execution boundaries.
func TaskUsesOperationsLane(task Task) bool {
	return IsOperationsTask(task)
}

func OperationsTaskIntent(task Task) string {
	intent := strings.TrimSpace(task.Intent)
	low := strings.ToLower(intent)
	if idx := strings.Index(low, RelayOperationsIntentPrefix); idx >= 0 && idx < 160 {
		intent = strings.TrimSpace(intent[:idx] + intent[idx+len(RelayOperationsIntentPrefix):])
	}
	return intent
}
