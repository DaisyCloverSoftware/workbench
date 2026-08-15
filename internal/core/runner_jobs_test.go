package core

import (
	"context"
	"strings"
	"testing"
)

func runnerJobTestRequest() RunnerRequest {
	return RunnerRequest{
		Task: Task{
			ID:          "task-durable-test",
			Intent:      "Make the requested change",
			ProjectPath: "/runner/src/project",
		},
		AvoidWorkUsage: true,
	}
}

func TestRunnerRequestFingerprintIgnoresDesktopLifecycleFields(t *testing.T) {
	req := runnerJobTestRequest()
	before, err := runnerRequestFingerprint(req)
	if err != nil {
		t.Fatal(err)
	}
	req.Task.Status = TaskRunning
	req.Task.ProviderID = "workbench-runner"
	req.Task.RouteReason = "recovered route"
	req.Task.Attempts = []string{"Workbench restarted; resuming task"}
	req.Task.ConsumesWork = true
	afterRecovery, err := runnerRequestFingerprint(req)
	if err != nil {
		t.Fatal(err)
	}
	if afterRecovery != before {
		t.Fatalf("desktop recovery fields changed durable request identity: %s != %s", afterRecovery, before)
	}

	req.Task.HumanAnswer = "Choose A"
	afterAnswer, err := runnerRequestFingerprint(req)
	if err != nil {
		t.Fatal(err)
	}
	if afterAnswer == before {
		t.Fatal("human attention answer must create a new durable request generation")
	}
}

func TestRunnerJobSubmitIsIdempotentForSameRequest(t *testing.T) {
	t.Setenv("WORKBENCH_RUNNER_JOB_ROOT", t.TempDir())
	req := runnerJobTestRequest()
	launches := 0
	launcher := func(string) (int, error) {
		launches++
		return 0, nil
	}

	first, err := submitRunnerJob(req, launcher)
	if err != nil {
		t.Fatal(err)
	}
	second, err := submitRunnerJob(req, launcher)
	if err != nil {
		t.Fatal(err)
	}
	if first.Reused {
		t.Fatal("first submit unexpectedly reported reuse")
	}
	if !second.Reused {
		t.Fatal("same request should reconnect to the existing durable job")
	}
	if launches != 1 {
		t.Fatalf("launcher called %d times, want 1", launches)
	}
	if second.Job.ID != req.Task.ID || second.Job.Generation != 1 {
		t.Fatalf("unexpected durable job identity: %+v", second.Job)
	}
}

func TestRunnerJobRejectsDifferentRequestWhileActive(t *testing.T) {
	t.Setenv("WORKBENCH_RUNNER_JOB_ROOT", t.TempDir())
	req := runnerJobTestRequest()
	launcher := func(string) (int, error) { return 0, nil }
	if _, err := submitRunnerJob(req, launcher); err != nil {
		t.Fatal(err)
	}
	req.Task.Intent = "A materially different request"
	if _, err := submitRunnerJob(req, launcher); err == nil || !strings.Contains(err.Error(), "already active") {
		t.Fatalf("expected active-job rejection, got %v", err)
	}
}

func TestRunnerJobExecutionPersistsCompletedResponse(t *testing.T) {
	t.Setenv("WORKBENCH_RUNNER_JOB_ROOT", t.TempDir())
	req := runnerJobTestRequest()
	if _, err := submitRunnerJob(req, func(string) (int, error) { return 0, nil }); err != nil {
		t.Fatal(err)
	}
	if err := executeStoredRunnerJob(req.Task.ID, func(context.Context, RunnerRequest) RunnerResponse {
		return RunnerResponse{Result: RunResult{Output: "done"}, ProviderID: "claude", ProviderName: "Claude"}
	}); err != nil {
		t.Fatal(err)
	}
	job, err := GetRunnerJob(req.Task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != RunnerJobCompleted || job.Response == nil {
		t.Fatalf("unexpected terminal job: %+v", job)
	}
	if job.Response.Result.Output != "done" || job.Response.ProviderID != "claude" {
		t.Fatalf("response was not persisted: %+v", job.Response)
	}
	if job.StartedAt == nil || job.FinishedAt == nil {
		t.Fatalf("job timestamps incomplete: %+v", job)
	}
}

func TestRunnerJobFastCompletionDoesNotRestoreStaleLauncherPID(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WORKBENCH_RUNNER_JOB_ROOT", root)
	req := runnerJobTestRequest()
	_, err := submitRunnerJob(req, func(id string) (int, error) {
		if err := executeStoredRunnerJob(id, func(context.Context, RunnerRequest) RunnerResponse {
			return RunnerResponse{Result: RunResult{Output: "finished immediately"}}
		}); err != nil {
			return 0, err
		}
		return 999999, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	record, found, err := loadRunnerJobRecord(root, req.Task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !found || record.Job.Status != RunnerJobCompleted {
		t.Fatalf("unexpected record after immediate completion: %+v", record)
	}
	if record.PID != 0 {
		t.Fatalf("terminal runner job retained stale launcher pid %d", record.PID)
	}
}

func TestRunnerJobAttentionCanResumeAsNewGeneration(t *testing.T) {
	t.Setenv("WORKBENCH_RUNNER_JOB_ROOT", t.TempDir())
	req := runnerJobTestRequest()
	launcher := func(string) (int, error) { return 0, nil }
	if _, err := submitRunnerJob(req, launcher); err != nil {
		t.Fatal(err)
	}
	if err := executeStoredRunnerJob(req.Task.ID, func(context.Context, RunnerRequest) RunnerResponse {
		return RunnerResponse{Result: RunResult{Attention: "Choose A or B"}, ProviderID: "claude"}
	}); err != nil {
		t.Fatal(err)
	}
	waiting, err := GetRunnerJob(req.Task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if waiting.Status != RunnerJobNeedsAttention {
		t.Fatalf("status=%s, want needs_attention", waiting.Status)
	}

	req.Task.HumanAnswer = "Choose A"
	resumed, err := submitRunnerJob(req, launcher)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Reused || resumed.Job.Generation != 2 || resumed.Job.Status != RunnerJobQueued {
		t.Fatalf("unexpected resumed generation: %+v", resumed)
	}
}

func TestCancelRunnerJobPersistsCancellation(t *testing.T) {
	t.Setenv("WORKBENCH_RUNNER_JOB_ROOT", t.TempDir())
	req := runnerJobTestRequest()
	if _, err := submitRunnerJob(req, func(string) (int, error) { return 0, nil }); err != nil {
		t.Fatal(err)
	}
	job, err := CancelRunnerJob(req.Task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != RunnerJobCancelled || job.FinishedAt == nil {
		t.Fatalf("unexpected cancelled job: %+v", job)
	}
	loaded, err := GetRunnerJob(req.Task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != RunnerJobCancelled {
		t.Fatalf("persisted status=%s, want cancelled", loaded.Status)
	}
}
