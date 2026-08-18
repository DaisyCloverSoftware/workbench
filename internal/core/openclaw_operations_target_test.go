package core

import (
	"reflect"
	"testing"
)

func TestOpenClawOperationAgentArgsSelectMainAgent(t *testing.T) {
	got := openClawOperationAgentArgs("verify runtime health")
	want := []string{"agent", "--agent", "main", "--message", "verify runtime health"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("operation args=%q want %q", got, want)
	}
}

func TestOpenClawOperationsAgentTargetIsExplicit(t *testing.T) {
	args := openClawOperationAgentArgs("test")
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "--agent" && args[i+1] != "" {
			return
		}
	}
	t.Fatalf("operations invocation has no explicit OpenClaw agent target: %q", args)
}
