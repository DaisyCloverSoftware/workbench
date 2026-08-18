package core

import (
	"errors"
	"strings"
	"testing"
)

func TestOpenClawOperationsPromptEnforcesOperatorBoundaryAndContinuation(t *testing.T) {
	task := Task{
		Intent:      "[relay:op_001] [workbench:operations] Restart the runner and verify it is healthy",
		ProjectPath: "/tmp/project",
	}
	prompt := BuildOpenClawOperationPrompt(task, 2, "service restarted but verification not finished")
	for _, want := range []string{
		"ChatGPT owns the software-development loop",
		"source code, Git/GitHub changes, pull requests, CI and GitHub Actions",
		"Do not implement application features",
		"do not mutate Git or GitHub state",
		"shell/systemd/Docker/Kubernetes/Helm",
		"do not stop merely to report progress",
		"WORKBENCH_OPERATION_COMPLETE: verified",
		"continuation pass 2 of 6",
		"Restart the runner and verify it is healthy",
	} {
		if !strings.Contains(strings.ToLower(prompt), strings.ToLower(want)) {
			t.Fatalf("operations prompt missing %q:\n%s", want, prompt)
		}
	}
	if strings.Contains(prompt, "[workbench:operations]") {
		t.Fatalf("relay control marker leaked into operator objective: %s", prompt)
	}
}

func TestStripOperationCompletionMarker(t *testing.T) {
	clean, complete := stripOperationCompletionMarker("Restarted service.\nVerified health.\nWORKBENCH_OPERATION_COMPLETE: verified")
	if !complete {
		t.Fatal("completion marker was not recognised")
	}
	if clean != "Restarted service.\nVerified health." {
		t.Fatalf("clean output=%q", clean)
	}
}

func TestOperationInvocationReengagesOnlyBoundedStall(t *testing.T) {
	if !operationInvocationCanBeReengaged(RunResult{Retryable: true}, errors.New("OpenClaw operations invocation timed out")) {
		t.Fatal("timeout should be re-engaged automatically")
	}
	if operationInvocationCanBeReengaged(RunResult{Retryable: true, Authentication: true}, errors.New("authentication failed")) {
		t.Fatal("authentication failure must not be hammered by supervisor")
	}
	if operationInvocationCanBeReengaged(RunResult{Attention: "production permission?"}, errors.New("blocked")) {
		t.Fatal("human attention must pause rather than auto-reengage")
	}
}
