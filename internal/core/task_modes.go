package core

import "strings"

const RelayOperationsIntentPrefix = "[workbench:operations]"

// TaskUsesOperationsLane recognises both first-class operations tasks and the
// backwards-compatible private-relay marker. The Git relay prepends its own
// [relay:<id>] trace tag, so accept the operations marker within the bounded
// leading control-tag region rather than requiring it to be byte zero.
func TaskUsesOperationsLane(task Task) bool {
	if IsOperationsTask(task) {
		return true
	}
	intent := strings.ToLower(strings.TrimSpace(task.Intent))
	idx := strings.Index(intent, RelayOperationsIntentPrefix)
	return idx >= 0 && idx < 160
}

func OperationsTaskIntent(task Task) string {
	intent := strings.TrimSpace(task.Intent)
	low := strings.ToLower(intent)
	if idx := strings.Index(low, RelayOperationsIntentPrefix); idx >= 0 && idx < 160 {
		intent = strings.TrimSpace(intent[:idx] + intent[idx+len(RelayOperationsIntentPrefix):])
	}
	return intent
}
