package core

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestOpenClawOperationsPromptEnforcesOperatorBoundaryAndContinuation(t *testing.T) {
	task := Task{
		ID:          "task-operation-001",
		Intent:      "[relay:op_001] [workbench:operations] Restart the runner and verify it is healthy",
		ProjectPath: "/tmp/project",
	}
	prompt := BuildOpenClawOperationPrompt(task, 2, "service restarted but verification not finished")
	for _, want := range []string{
		"owner explicitly assigned to OpenClaw by name",
		"ChatGPT owns the software-development loop",
		"source code, Git/GitHub changes, pull requests, CI, GitHub Actions, releases",
		"Do not implement application features",
		"do not mutate Git or GitHub state",
		"shell/systemd/Docker/Kubernetes/Helm",
		"Do not infer or expand authority",
		"do not stop merely to report progress",
		"WORKBENCH_OPERATION_COMPLETE: verified",
		"continuation pass 2 of 6",
		"same Workbench job conversation",
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

func TestOpenClawOperationAgentArgsUseScriptedJobSession(t *testing.T) {
	task := Task{ID: "task-operation-001"}
	prompt := "verify runner health"
	sessionID := openClawOperationSessionID(task)
	got := openClawOperationAgentArgsWithSession(prompt, "", sessionID)
	want := []string{"agent", "--agent", "main", "--session-id", sessionID, "--message", prompt}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("OpenClaw agent args=%q want %q", got, want)
	}
	for _, arg := range got {
		if arg == "--headless" {
			t.Fatal("OpenClaw agent does not accept the browser-only --headless flag")
		}
	}
}

func TestOpenClawOperationSessionIsStablePerJobAndDifferentAcrossJobs(t *testing.T) {
	firstJob := Task{ID: "task-operation-001"}
	secondJob := Task{ID: "task-operation-002"}
	first := openClawOperationSessionID(firstJob)
	continued := openClawOperationSessionID(firstJob)
	second := openClawOperationSessionID(secondJob)
	if !strings.HasPrefix(first, "workbench-op-") {
		t.Fatalf("operation session id=%q", first)
	}
	if first != continued {
		t.Fatalf("same Workbench job changed OpenClaw conversation: %q != %q", first, continued)
	}
	if first == second {
		t.Fatalf("different Workbench jobs shared OpenClaw conversation %q", first)
	}
	if got := openClawOperationSessionKey(firstJob); got != "agent:main:explicit:"+first {
		t.Fatalf("session key=%q", got)
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
		t.Fatal("timeout should be re-engaged automatically in the same job conversation")
	}
	if operationInvocationCanBeReengaged(RunResult{Retryable: true}, errors.New("Codex binding generation was retired: session-key:main:deadbeef")) {
		t.Fatal("a corrupted job conversation should fail rather than loop forever on the same retired binding")
	}
	if operationInvocationCanBeReengaged(RunResult{Retryable: true, Authentication: true}, errors.New("authentication failed")) {
		t.Fatal("authentication failure must not be hammered by supervisor")
	}
	if operationInvocationCanBeReengaged(RunResult{Attention: "production permission?"}, errors.New("blocked")) {
		t.Fatal("human attention must pause rather than auto-reengage")
	}
}
