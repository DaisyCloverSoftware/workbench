package core

import "strings"

const RelayOperationsIntentPrefix = "[workbench:operations]"

// TaskUsesOperationsLane recognises both first-class operations tasks and the
// backwards-compatible private-relay marker. The relay marker lets an upgraded
// private Git bridge use the operations lane even while an older desktop MCP
// only exposes delegate_task.
func TaskUsesOperationsLane(task Task) bool {
	if IsOperationsTask(task) {
		return true
	}
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(task.Intent)), RelayOperationsIntentPrefix)
}

func OperationsTaskIntent(task Task) string {
	intent := strings.TrimSpace(task.Intent)
	if strings.HasPrefix(strings.ToLower(intent), RelayOperationsIntentPrefix) {
		intent = strings.TrimSpace(intent[len(RelayOperationsIntentPrefix):])
	}
	return intent
}
