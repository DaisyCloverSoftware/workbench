package core

import (
	"reflect"
	"testing"
)

func TestOpenClawOperationAgentArgsSelectMainAgent(t *testing.T) {
	got := openClawOperationAgentArgsWithSession("verify runtime health", "", "openclaw-op-test")
	want := []string{"agent", "--agent", "main", "--session-id", "openclaw-op-test", "--message", "verify runtime health"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("operation args=%q want %q", got, want)
	}
}

func TestOpenClawOperationsAgentTargetAndSessionAreExplicit(t *testing.T) {
	args := openClawOperationAgentArgs("test")
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
		t.Fatalf("operations invocation has no explicit isolated session target: %q", args)
	}
}
