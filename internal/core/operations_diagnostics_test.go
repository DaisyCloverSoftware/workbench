package core

import (
	"errors"
	"strings"
	"testing"
)

func TestOperationFailureDetailReturnsBoundedSafeOutput(t *testing.T) {
	task := Task{Mode: TaskModeOperations}
	res := RunResult{Output: "OpenClaw runtime failed:\nnode executable not found by service"}
	got := operationFailureDetail(task, res, errors.New("exit status 1"))
	if got != "OpenClaw runtime failed: node executable not found by service" {
		t.Fatalf("detail=%q", got)
	}

	res.Output = strings.Repeat("x", maxOperationFailureDetailRunes+200)
	got = operationFailureDetail(task, res, errors.New("exit status 1"))
	if len([]rune(got)) != maxOperationFailureDetailRunes+1 || !strings.HasSuffix(got, "…") {
		t.Fatalf("bounded detail length/suffix unexpected: runes=%d suffix=%q", len([]rune(got)), got[len(got)-3:])
	}
}

func TestOperationFailureDetailWithholdsSecretLikeOutput(t *testing.T) {
	task := Task{Mode: TaskModeOperations}
	secret := "worker echoed credential: " + "sk-" + "proj-" + strings.Repeat("x", 48)
	if got := operationFailureDetail(task, RunResult{Output: secret}, errors.New("exit status 1")); got != "" {
		t.Fatalf("secret-like operation output must be withheld: %q", got)
	}
	if got := operationFailureDetail(Task{}, RunResult{Output: "ordinary coding failure"}, errors.New("exit status 1")); got != "" {
		t.Fatalf("development task diagnostics must not be copied into the operations detail path: %q", got)
	}
}

func TestOperationsRoutingCannotFallThroughToCodingWorkers(t *testing.T) {
	providers := []Provider{
		{ID: "workbench-runner", Name: "Runner", Installed: true, Authenticated: true, CanWrite: true, Command: "ssh", Cost: CostIncluded, Priority: 10},
		{ID: "openclaw", Name: "OpenClaw", Installed: true, Authenticated: true, CanWrite: true, Command: "/tmp/openclaw", Cost: CostIncluded, Priority: 20},
		{ID: "claude", Name: "Claude", Installed: true, Authenticated: true, CanWrite: true, Command: "claude", Cost: CostIncluded, Priority: 30},
		{ID: "codex", Name: "Codex", Installed: true, Authenticated: true, CanWrite: true, Command: "codex", Cost: CostScarce, Priority: 40},
	}
	prefs := Preferences{OpenClawSSHHost: "runner.example"}

	unauthorized := routeCandidates(providers, prefs, Task{Mode: TaskModeOperations, ProjectPath: t.TempDir()})
	if len(unauthorized) != 0 {
		t.Fatalf("unauthorized Operations task received candidates=%+v", unauthorized)
	}

	local := routeCandidates(providers, prefs, Task{Mode: TaskModeOperations, OpenClawOwnerAuthorized: true, ProjectPath: t.TempDir()})
	if len(local) != 1 || local[0].ID != "openclaw" {
		t.Fatalf("explicitly authorized local operations candidates=%+v; want OpenClaw only", local)
	}

	remote := routeCandidates(providers, prefs, Task{Mode: TaskModeOperations, OpenClawOwnerAuthorized: true, ProjectPath: "runner://workbench"})
	if len(remote) != 1 || remote[0].ID != "workbench-runner" {
		t.Fatalf("explicitly authorized runner operations candidates=%+v; want Workbench runner only", remote)
	}
}
