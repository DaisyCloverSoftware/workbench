package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	openClawOperationFallbackModelEnv = "WORKBENCH_OPENCLAW_OPERATION_FALLBACK_MODEL"
	openClawFallbackProbeTimeout      = 10 * time.Second
)

// runOpenClawOperationInvocationWithFallback keeps the configured OpenClaw
// agent/model as the primary operations worker, but prevents a provider quota
// window from turning into a human babysitting problem. If the primary agent
// process fails specifically because model capacity/usage is exhausted,
// Workbench retries the same bounded operational prompt against an explicitly
// configured fallback model or, when available, a suitable local Ollama model.
func runOpenClawOperationInvocationWithFallback(ctx context.Context, p Provider, task Task, prefs Preferences, prompt string) (RunResult, bool, error) {
	res, complete, err := runOpenClawOperationInvocation(ctx, p, task, prefs, prompt)
	if err == nil || !operationModelCapacityFailure(res, err) || ctx.Err() != nil {
		return res, complete, err
	}

	model := strings.TrimSpace(os.Getenv(openClawOperationFallbackModelEnv))
	if model == "" {
		model = detectOpenClawOperationsLocalModel(ctx, prefs)
	}
	if model == "" {
		return res, complete, err
	}

	fallbackRes, fallbackComplete, fallbackErr := runOpenClawOperationModelOverride(ctx, p, task, prefs, prompt, model)
	if fallbackErr == nil {
		return fallbackRes, fallbackComplete, nil
	}
	fallbackRes.Retryable = true
	if strings.TrimSpace(fallbackRes.Output) == "" {
		fallbackRes.Output = "OpenClaw's primary model was unavailable and the detected local operations fallback could not complete the task."
	}
	return boundRunResultForPersistence(fallbackRes), false, fmt.Errorf("OpenClaw primary model capacity was exhausted and fallback model %s also failed: %w", model, fallbackErr)
}

func operationModelCapacityFailure(res RunResult, err error) bool {
	if err == nil || res.Authentication || strings.TrimSpace(res.Attention) != "" || strings.TrimSpace(res.WorkerUnavailable) != "" {
		return false
	}
	low := strings.ToLower(err.Error() + " " + res.Output)
	for _, marker := range []string{
		"usage limit",
		"quota",
		"rate limit",
		"rate_limit",
		"billing",
		"credit limit",
		"credits exhausted",
		"capacity exhausted",
		"too many requests",
		"subscription usage",
	} {
		if strings.Contains(low, marker) {
			return true
		}
	}
	return false
}

// detectOpenClawOperationsLocalModel asks only the local Ollama inventory and
// never reads model files or credentials. It deliberately prefers compact
// tool/coding-capable families that are plausible infrastructure operators.
func detectOpenClawOperationsLocalModel(ctx context.Context, prefs Preferences) string {
	probeCtx, cancel := context.WithTimeout(ctx, openClawFallbackProbeTimeout)
	defer cancel()

	remote := strings.TrimSpace(prefs.OpenClawSSHHost) != ""
	name := "ollama"
	args := []string{"list"}
	if remote {
		name = "ssh"
		args = []string{
			"-o", "BatchMode=yes",
			"-o", "ConnectTimeout=10",
			"-o", "StrictHostKeyChecking=accept-new",
			strings.TrimSpace(prefs.OpenClawSSHHost),
			"ollama", "list",
		}
	}

	cmd := exec.CommandContext(probeCtx, name, args...)
	if !remote {
		cmd.Env = environmentForUserExecutable(name)
	}
	configureChildProcess(cmd, false)
	stdout := newBoundedWorkerCapture(64 << 10)
	stderr := newBoundedWorkerCapture(16 << 10)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil || stdout.Truncated() || stderr.Truncated() {
		return ""
	}
	model := preferredOllamaOperationsModel(stdout.String())
	if model == "" {
		return ""
	}
	return "ollama/" + model
}

func preferredOllamaOperationsModel(listOutput string) string {
	type candidate struct {
		name  string
		rank  int
		order int
	}
	best := candidate{rank: 1 << 30, order: 1 << 30}
	order := 0
	for _, line := range strings.Split(listOutput, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) == 0 {
			continue
		}
		name := strings.TrimSpace(fields[0])
		low := strings.ToLower(name)
		if low == "name" || strings.Contains(low, "embed") || strings.Contains(low, "nomic") || strings.Contains(low, "bge-") {
			continue
		}

		rank := 100
		switch {
		case strings.Contains(low, "qwen") && strings.Contains(low, "coder"):
			rank = 0
		case strings.Contains(low, "qwen"):
			rank = 10
		case strings.Contains(low, "coder") || strings.Contains(low, "code"):
			rank = 20
		case strings.Contains(low, "llama"):
			rank = 30
		case strings.Contains(low, "mistral") || strings.Contains(low, "gemma"):
			rank = 40
		default:
			continue
		}
		if rank < best.rank || (rank == best.rank && order < best.order) {
			best = candidate{name: name, rank: rank, order: order}
		}
		order++
	}
	return best.name
}

func openClawOperationAgentArgsForModel(prompt, model string) []string {
	return openClawOperationAgentArgsWithSession(prompt, model, newOpenClawOperationSessionID())
}

func runOpenClawOperationModelOverride(ctx context.Context, p Provider, task Task, prefs Preferences, prompt, model string) (RunResult, bool, error) {
	invokeCtx, cancel := context.WithTimeout(ctx, operationInvocationTimeout)
	defer cancel()

	var name string
	var args []string
	remote := strings.TrimSpace(prefs.OpenClawSSHHost) != ""
	if remote {
		name = "ssh"
		args = []string{"-o", "BatchMode=yes", "-o", "ConnectTimeout=10", "-o", "StrictHostKeyChecking=accept-new", strings.TrimSpace(prefs.OpenClawSSHHost), "openclaw"}
		args = append(args, openClawOperationAgentArgsForModel(prompt, model)...)
	} else if strings.TrimSpace(p.Command) != "" {
		name = p.Command
		args = openClawOperationAgentArgsForModel(prompt, model)
	} else {
		return RunResult{Retryable: true}, false, errors.New("OpenClaw operations adapter is not configured")
	}

	cmd := exec.CommandContext(invokeCtx, name, args...)
	if !remote {
		cmd.Dir = task.ProjectPath
		cmd.Env = environmentForUserExecutable(name)
	}
	configureChildProcess(cmd, false)
	stdout := newBoundedWorkerCapture(maxWorkerStreamCaptureBytes)
	stderr := newBoundedWorkerCapture(maxWorkerStreamCaptureBytes)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	runErr := cmd.Run()

	out := strings.TrimSpace(stdout.String())
	if se := strings.TrimSpace(stderr.String()); se != "" {
		if out != "" {
			out += "\n\n"
		}
		out += se
	}
	out = normalizeProviderOutput("openclaw", out)
	clean, complete := stripOperationCompletionMarker(out)
	res := classifyRunOutput(clean)

	if stdout.Truncated() || stderr.Truncated() {
		res.Retryable = true
		return boundRunResultForPersistence(res), false, errors.New("OpenClaw fallback operations response exceeded Workbench's bounded output limit")
	}
	if strings.TrimSpace(res.WorkerUnavailable) != "" {
		res.Retryable = true
		return boundRunResultForPersistence(res), false, fmt.Errorf("OpenClaw fallback is unavailable for this operational task: %s", res.WorkerUnavailable)
	}
	if strings.TrimSpace(res.Attention) != "" && isWorkerSetupAttention(res.Attention) {
		q := res.Attention
		res.Attention = ""
		res.Retryable = true
		return boundRunResultForPersistence(res), false, fmt.Errorf("OpenClaw fallback cannot act autonomously under its current local tool permissions: %s", q)
	}
	if runErr != nil {
		if errors.Is(invokeCtx.Err(), context.DeadlineExceeded) && ctx.Err() == nil {
			res.Retryable = true
			if strings.TrimSpace(res.Output) == "" {
				res.Output = "OpenClaw local fallback became unresponsive before confirming completion."
			}
			return boundRunResultForPersistence(res), false, errors.New("OpenClaw fallback operations invocation timed out")
		}
		low := strings.ToLower(out + " " + runErr.Error())
		res.Authentication = strings.Contains(low, "login") || strings.Contains(low, "sign in") || strings.Contains(low, "authenticate") || strings.Contains(low, "unauthorized") || strings.Contains(low, "credential") || strings.Contains(low, "publickey") || strings.Contains(low, "permission denied")
		res.Retryable = true
		if strings.TrimSpace(res.Output) == "" {
			res.Output = runErr.Error()
		}
		return boundRunResultForPersistence(res), false, fmt.Errorf("OpenClaw fallback operations invocation exited with error: %w", runErr)
	}
	return boundRunResultForPersistence(res), complete, nil
}
