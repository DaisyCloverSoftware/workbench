package core

import "testing"

func TestRelayOperationsMarkerSurvivesRelayTraceTag(t *testing.T) {
	task := Task{Intent: "[relay:cluster_op_001] [workbench:operations] Deploy DEV and verify rollout"}
	if !IsOperationsTask(task) || !TaskUsesOperationsLane(task) {
		t.Fatalf("relay-tagged task was not recognised as operations: %#v", task)
	}
	if got := OperationsTaskIntent(task); got != "[relay:cluster_op_001]  Deploy DEV and verify rollout" {
		t.Fatalf("operations intent=%q", got)
	}
}

func TestFirstClassOperationsModeIsRecognised(t *testing.T) {
	task := Task{Mode: TaskModeOperations, Intent: "restart runner"}
	if !IsOperationsTask(task) {
		t.Fatal("first-class operations mode was not recognised")
	}
}

func TestOrdinaryDevelopmentIntentDoesNotEnterOperationsLane(t *testing.T) {
	task := Task{Intent: "Fix the Workbench dashboard"}
	if IsOperationsTask(task) {
		t.Fatal("ordinary development task entered operations lane")
	}
}
