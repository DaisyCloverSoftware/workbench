package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// RunnerRequest is the structured task envelope sent from a Workbench desktop
// to a Workbench Runner on another machine. The runner re-routes locally so the
// desktop does not need to know which coding CLIs are installed on that host.
type RunnerRequest struct {
	Task            Task `json:"task"`
	AvoidWorkUsage  bool `json:"avoid_work_usage"`
	AllowMeteredAPI bool `json:"allow_metered_api"`
}

// RunnerResponse is deliberately small and transport-neutral. A future HTTP or
// message-queue transport can use exactly the same envelope as the SSH adapter.
type RunnerResponse struct {
	Result       RunResult `json:"result"`
	ProviderID   string    `json:"provider_id,omitempty"`
	ProviderName string    `json:"provider_name,omitempty"`
	ProviderCost CostClass `json:"provider_cost,omitempty"`
	Error        string    `json:"error,omitempty"`
	Attempts     []string  `json:"attempts,omitempty"`
}

type runnerJobSubmitEnvelope struct {
	OK     bool                  `json:"ok"`
	Result RunnerJobSubmitResult `json:"result"`
	Error  string                `json:"error,omitempty"`
}

type runnerJobStatusEnvelope struct {
	OK    bool      `json:"ok"`
	Job   RunnerJob `json:"job"`
	Error string    `json:"error,omitempty"`
}

// RunClusterRunnerSSH submits a durable job to the cluster-side Workbench
// Runner, then polls that job over short independent SSH calls. The worker is
// deliberately detached from the submitting SSH session, so closing/restarting
// desktop Workbench or losing a network connection does not discard work. A
// later submit of the exact same task request reconnects to the existing job.
func RunClusterRunnerSSH(ctx context.Context, host string, task Task, prefs Preferences) (RunResult, error) {
	validatedHost, err := validateSSHHostTarget(host)
	if err != nil {
		return RunResult{Retryable: true}, err
	}
	host = validatedHost
	req := RunnerRequest{
		Task:            task,
		AvoidWorkUsage:  prefs.AvoidWorkUsage,
		AllowMeteredAPI: prefs.AllowMeteredAPI,
	}
	reqBody, err := json.Marshal(req)
	if err != nil {
		return RunResult{}, err
	}

	var submitted RunnerJob
	for {
		stdout, stderr, truncated, runErr := runRunnerSSHCommand(ctx, host, reqBody, "job", "submit")
		combined := combineRunnerSSHOutput(stdout, stderr)
		if runErr != nil {
			if ctx.Err() != nil {
				// The remote side may have accepted the idempotent submit before
				// the local SSH process was cancelled. Explicit cancellation must
				// therefore attempt the task ID even without a submit response.
				cancelRunnerJobBestEffort(host, task.ID)
				return RunResult{}, ctx.Err()
			}
			if durableJobCommandUnsupported(combined) {
				return runClusterRunnerLegacySSH(ctx, host, reqBody)
			}
			if runnerSSHAuthenticationFailure(combined, runErr) {
				res := classifyRunOutput(combined)
				res.Retryable = true
				res.Authentication = true
				return boundRunResultForPersistence(res), fmt.Errorf("cluster runner SSH authentication failed: %w", runErr)
			}
			if !waitRunnerRetry(ctx, 2*time.Second) {
				cancelRunnerJobBestEffort(host, task.ID)
				return RunResult{}, ctx.Err()
			}
			continue
		}
		if truncated {
			// The runner may have accepted the idempotent task before its reply
			// exceeded our transport cap. Retry the same task ID instead of
			// returning an error that could route another worker onto the repo.
			if !waitRunnerRetry(ctx, 2*time.Second) {
				cancelRunnerJobBestEffort(host, task.ID)
				return RunResult{}, ctx.Err()
			}
			continue
		}
		var envelope runnerJobSubmitEnvelope
		if err := json.Unmarshal(stdout, &envelope); err != nil {
			if durableJobCommandUnsupported(combined) {
				return runClusterRunnerLegacySSH(ctx, host, reqBody)
			}
			// A syntactically broken acknowledgement is also an ambiguous submit:
			// the job may exist. Re-submit idempotently rather than fail over.
			if !waitRunnerRetry(ctx, 2*time.Second) {
				cancelRunnerJobBestEffort(host, task.ID)
				return RunResult{}, ctx.Err()
			}
			continue
		}
		if !envelope.OK {
			return RunResult{Output: boundPersistedWorkerText(envelope.Error)}, errors.New(strings.TrimSpace(envelope.Error))
		}
		submitted = envelope.Result.Job
		if submitted.ID != task.ID {
			return RunResult{}, errors.New("cluster runner durable job identity mismatch")
		}
		break
	}

	for {
		select {
		case <-ctx.Done():
			cancelRunnerJobBestEffort(host, submitted.ID)
			return RunResult{}, ctx.Err()
		default:
		}

		stdout, _, truncated, runErr := runRunnerSSHCommand(ctx, host, nil, "job", "status", submitted.ID)
		if runErr != nil || truncated {
			// The durable job may still be running even if the SSH account,
			// network/host is temporarily unreachable or its response is too
			// large. Returning an error here would make Engine hand the same
			// repository to another coding worker. Stay attached logically until
			// the task context is explicitly cancelled or expires.
			if !waitRunnerRetry(ctx, 2*time.Second) {
				cancelRunnerJobBestEffort(host, submitted.ID)
				return RunResult{}, ctx.Err()
			}
			continue
		}
		var envelope runnerJobStatusEnvelope
		if err := json.Unmarshal(stdout, &envelope); err != nil {
			// A known durable job is safer to poll again than to fail over on a
			// malformed/transient status response.
			if !waitRunnerRetry(ctx, 2*time.Second) {
				cancelRunnerJobBestEffort(host, submitted.ID)
				return RunResult{}, ctx.Err()
			}
			continue
		}
		if !envelope.OK {
			return RunResult{Output: boundPersistedWorkerText(envelope.Error)}, errors.New(strings.TrimSpace(envelope.Error))
		}
		job := envelope.Job
		if job.ID != submitted.ID {
			return RunResult{}, errors.New("cluster runner durable job status identity mismatch")
		}
		switch job.Status {
		case RunnerJobQueued, RunnerJobRunning:
			if !waitRunnerRetry(ctx, time.Second) {
				cancelRunnerJobBestEffort(host, submitted.ID)
				return RunResult{}, ctx.Err()
			}
			continue
		case RunnerJobNeedsAttention, RunnerJobCompleted:
			if job.Response == nil {
				return RunResult{}, errors.New("cluster runner durable job is terminal without a response")
			}
			return runResultFromRunnerResponse(*job.Response)
		case RunnerJobFailed:
			if job.Response != nil {
				res, responseErr := runResultFromRunnerResponse(*job.Response)
				if responseErr != nil {
					return res, responseErr
				}
				if strings.TrimSpace(job.Error) != "" {
					return res, errors.New(job.Error)
				}
				return res, errors.New("cluster runner durable job failed")
			}
			return RunResult{Output: boundPersistedWorkerText(job.Error)}, errors.New(strings.TrimSpace(job.Error))
		case RunnerJobCancelled:
			return RunResult{}, context.Canceled
		default:
			return RunResult{}, fmt.Errorf("cluster runner returned unknown durable job status %q", job.Status)
		}
	}
}

func runResultFromRunnerResponse(rr RunnerResponse) (RunResult, error) {
	res := boundRunResultForPersistence(rr.Result)
	res.WorkerProviderID = rr.ProviderID
	res.WorkerProviderName = rr.ProviderName
	res.WorkerCost = rr.ProviderCost
	if strings.TrimSpace(rr.Error) != "" {
		res.Retryable = true
		return res, errors.New(boundWorkerControlText(rr.Error))
	}
	return res, nil
}

func runRunnerSSHCommand(ctx context.Context, host string, stdin []byte, args ...string) ([]byte, []byte, bool, error) {
	remoteArgs := []string{"$HOME/.local/bin/workbench-runner"}
	remoteArgs = append(remoteArgs, args...)
	cmdArgs := []string{
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=10",
		"-o", "StrictHostKeyChecking=accept-new",
		host,
	}
	cmdArgs = append(cmdArgs, remoteArgs...)
	cmd := exec.CommandContext(ctx, "ssh", cmdArgs...)
	configureChildProcess(cmd, false)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	stdout := newBoundedWorkerCapture(maxWorkerStreamCaptureBytes)
	stderr := newBoundedWorkerCapture(maxWorkerStreamCaptureBytes)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	return []byte(stdout.String()), []byte(stderr.String()), stdout.Truncated() || stderr.Truncated(), err
}

func combineRunnerSSHOutput(stdout, stderr []byte) string {
	combined := strings.TrimSpace(string(stdout))
	if se := strings.TrimSpace(string(stderr)); se != "" {
		if combined != "" {
			combined += "\n\n"
		}
		combined += se
	}
	return combined
}

func runnerSSHAuthenticationFailure(output string, err error) bool {
	low := strings.ToLower(output)
	if err != nil {
		low += " " + strings.ToLower(err.Error())
	}
	return strings.Contains(low, "permission denied") || strings.Contains(low, "publickey") || strings.Contains(low, "authentication failed")
}

func durableJobCommandUnsupported(output string) bool {
	low := strings.ToLower(output)
	return strings.Contains(low, "usage: workbench-runner") && !strings.Contains(low, "job <submit|status|cancel>")
}

func waitRunnerRetry(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func cancelRunnerJobBestEffort(host, jobID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, _, _, _ = runRunnerSSHCommand(ctx, host, nil, "job", "cancel", jobID)
}

// runClusterRunnerLegacySSH keeps rolling upgrades compatible with a runner
// older than the durable-job protocol. New runners never use this path.
func runClusterRunnerLegacySSH(ctx context.Context, host string, reqBody []byte) (RunResult, error) {
	stdout, stderr, truncated, runErr := runRunnerSSHCommand(ctx, host, reqBody, "run")
	combined := combineRunnerSSHOutput(stdout, stderr)
	if truncated {
		res := classifyRunOutput(combined)
		res.Retryable = true
		return boundRunResultForPersistence(res), errors.New("legacy cluster runner response exceeded Workbench's bounded transport limit")
	}
	var rr RunnerResponse
	decodeErr := json.Unmarshal(stdout, &rr)
	if decodeErr == nil {
		res, responseErr := runResultFromRunnerResponse(rr)
		if responseErr != nil {
			return res, responseErr
		}
		if runErr != nil {
			res.Retryable = true
			return res, runErr
		}
		return res, nil
	}
	res := classifyRunOutput(combined)
	res.Retryable = true
	if runErr != nil {
		res.Authentication = runnerSSHAuthenticationFailure(combined, runErr)
		if combined == "" {
			res.Output = runErr.Error()
		}
		return boundRunResultForPersistence(res), fmt.Errorf("cluster runner SSH failed: %w", runErr)
	}
	return boundRunResultForPersistence(res), fmt.Errorf("cluster runner returned invalid response: %w", decodeErr)
}

// ExecuteRunnerRequest is the cluster-side routing loop. It uses the same cost
// policy as the desktop and therefore prefers zero-marginal/included workers
// before scarce Work/Codex, while leaving metered APIs disabled unless opted in.
// Provider-level retryable failures are persisted in the local health cache so
// one-shot runner workers do not immediately hammer the same known-bad worker
// again on the next task.
func ExecuteRunnerRequest(ctx context.Context, req RunnerRequest) RunnerResponse {
	task := req.Task
	resolved, err := ResolveRunnerProject(task.ProjectPath)
	if err != nil {
		return RunnerResponse{Error: err.Error()}
	}
	task.ProjectPath = resolved
	prefs := Preferences{AvoidWorkUsage: req.AvoidWorkUsage, AllowMeteredAPI: req.AllowMeteredAPI}
	providers := ScanProviders()
	candidates := routeCandidates(providers, prefs, task)
	candidates, cooling := FilterProviderCooldowns(candidates, time.Now())

	attempts := append([]string(nil), cooling...)
	for _, p := range candidates {
		// A runner must never recursively delegate to another remote Workbench
		// runner through the same SSH configuration.
		if p.ID == "workbench-runner" {
			continue
		}
		res, runErr := RunProviderIsolated(ctx, p, task, prefs)
		res = boundRunResultForPersistence(res)
		record, coolingNow := RecordProviderRunOutcome(p.ID, res, runErr)
		attempt := fmt.Sprintf("%s: %s", p.Name, attemptSummary(res, runErr))
		if coolingNow {
			attempt += fmt.Sprintf("; cooldown until %s (%s)", record.CooldownUntil.UTC().Format(time.RFC3339), record.Reason)
		}
		attempts = append(attempts, attempt)
		if strings.TrimSpace(res.Attention) != "" {
			return RunnerResponse{Result: res, ProviderID: p.ID, ProviderName: p.Name, ProviderCost: p.Cost, Attempts: attempts}
		}
		if runErr == nil {
			return RunnerResponse{Result: res, ProviderID: p.ID, ProviderName: p.Name, ProviderCost: p.Cost, Attempts: attempts}
		}
	}
	if len(candidates) == 0 {
		if len(cooling) > 0 {
			return RunnerResponse{Error: "all eligible coding workers are temporarily cooling down after recent provider-level failures", Attempts: attempts}
		}
		return RunnerResponse{Error: "no eligible coding worker is installed on the runner host"}
	}
	return RunnerResponse{Error: "every eligible runner worker failed", Attempts: attempts}
}

func runnerRoot() (string, error) {
	if root := strings.TrimSpace(os.Getenv("WORKBENCH_RUNNER_ROOT")); root != "" {
		return filepath.Abs(root)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "src"), nil
}

// ResolveRunnerProject maps a desktop or runner-local project identifier to the
// repository directory under WORKBENCH_RUNNER_ROOT. Operator control commands
// use the same resolver as task execution so publication policy can be
// configured without knowing a different host-specific project path.
func ResolveRunnerProject(requested string) (string, error) {
	configuredRoot, err := runnerRoot()
	if err != nil {
		return "", err
	}
	root, err := canonicalRunnerDirectory(configuredRoot)
	if err != nil {
		return "", fmt.Errorf("runner root is not a directory: %s", configuredRoot)
	}

	requested = strings.TrimSpace(requested)
	if requested == "" {
		return "", errors.New("project path is empty")
	}

	// First accept a real path on the runner host, but only after resolving
	// symlinks so an in-root link cannot escape the authorised runner root.
	if abs, absErr := filepath.Abs(requested); absErr == nil {
		if resolved, resolveErr := canonicalRunnerDirectory(abs); resolveErr == nil {
			if withinRoot(root, resolved) {
				return resolved, nil
			}
			return "", fmt.Errorf("project is outside WORKBENCH_RUNNER_ROOT: %s", resolved)
		}
	}

	// Desktop paths are often Windows paths. Map their final directory name to
	// ~/src/<repo-name> on the runner. This makes a cloned repository portable
	// across Windows desktop + Linux cluster without hand-editing every task.
	normalized := strings.ReplaceAll(requested, "\\", "/")
	name := filepath.Base(strings.TrimRight(normalized, "/"))
	if name == "." || name == "/" || name == "" {
		return "", fmt.Errorf("cannot derive repository name from %q", requested)
	}
	candidate := filepath.Join(root, name)
	resolved, err := canonicalRunnerDirectory(candidate)
	if err != nil {
		return "", fmt.Errorf("runner cannot find project %q; expected %s", requested, candidate)
	}
	if !withinRoot(root, resolved) {
		return "", fmt.Errorf("project is outside WORKBENCH_RUNNER_ROOT: %s", resolved)
	}
	return resolved, nil
}

func canonicalRunnerDirectory(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return "", errors.New("path is not a directory")
	}
	return filepath.Clean(resolved), nil
}

func withinRoot(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
}
