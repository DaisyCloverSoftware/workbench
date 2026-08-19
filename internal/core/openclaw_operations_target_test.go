package core

import (
	"reflect"
	"testing"
)

func TestOpenClawOperationAgentArgsSelectMainAgent(t *testing.T) {
	task := Task{ID: "task-operation-001"}
	sessionID := openClawOperationSessionID(task)
	got := openClawOperationAgentArgs(task, "verify runtime health")
	want := []string{"agent", "--agent", "main", "--session-id", sessionID, "--message", "verify runtime health"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("operation args=%q want %q", got, want)
	}
}

func TestOpenClawOperationsAgentTargetAndSessionAreExplicit(t *testing.T) {
	args := openClawOperationAgentArgs(Task{ID: "task-operation-001"}, "test")
	agent := ""
	session := ""
	for i := 0; i+1 < len(args); i++ {
		switch args[i] {
		case "--agent":
			agent = args[i+1]
		case "--session-id":
			session = args[i+1]
		}
	}
	if agent == "" {
		t.Fatalf("operations invocation has no explicit OpenClaw agent target: %q", args)
	}
	if session == "" {
		t.Fatalf("operations invocation has no explicit job-scoped session target: %q", args)
	}
}
