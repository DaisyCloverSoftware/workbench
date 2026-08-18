package core

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	operationCompletePrefix        = "WORKBENCH_OPERATION_COMPLETE:"
	maxOperationContinuationPasses = 6
	operationInvocationTimeout      = 10 * time.Minute
	maxOperationContinuationReport = 4000
)

// RunOpenClawOperationSupervised is the missing "keep going" loop. A clean
// OpenClaw process exit is not considered task completion unless OpenClaw has
// explicitly verified the requested operational outcome. Progress-only exits
// and bounded unresponsive invocations are re-engaged automatically against the
// same real host/cluster state instead of making the human type "continue".
func RunOpenClawOperationSupervised(ctx context.Context, p Provider, task Task, prefs Preferences) (RunResult, error) {
	if p.ID != "openclaw" {
		return RunResult{}, errors.New("operations lane requires OpenClaw")
	}
	previous := ""
	for pass := 1; pass <= maxOperationContinuationPasses; pass++ {
		if err := ctx.Err(); err != nil {
			return RunResult{}, err
		}
		prompt := BuildOpenClawOperationPrompt(task, pass, previous)
		res, complete, runErr := runOpenClawOperationInvocation(ctx, p, task, prefs, prompt)
		if strings.TrimSpace(res.Attention) != "" || strings.TrimSpace(res.WorkerUnavailable) != "" {
			return boundRunResultForPersistence(res), runErr
		}
		if runErr != nil {
			if pass < maxOperationContinuationPasses && operationInvocationCanBeReengaged(res, runErr) {
				previous = operationContinuationReport(res.Output)
				if !waitRunnerRetry(ctx, 2*time.Second) {
					return boundRunResultForPersistence(res), ctx.Err()
				}
				continue
			}
			return boundRunResultForPersistence(res), runErr
		}
		if complete {
			return boundRunResultForPersistence(res), nil
		}
		previous = operationContinuationReport(res.Output)
	}
	res := RunResult{Output: previous, Retryable: true}
	return boundRunResultForPersistence(res), fmt.Errorf("OpenClaw stopped %d times without verifying the operational objective; Workbench exhausted its automatic continuation budget instead of asking the human to keep nudging it", maxOperationContinuationPasses)
}

func BuildOpenClawOperationPrompt(task Task, pass int, previous string) string {
	var b strings.Builder
	b.WriteString("You are OpenClaw acting only as Workbench's infrastructure/host/cluster operator. ChatGPT is the primary reasoning and coding brain. Your job is to execute operational work so the human never has to copy prompts here or keep telling you to continue.\n\n")
	b.WriteString("Operational objective:\n")
	b.WriteString(OperationsTaskIntent(task))
	b.WriteString("\n\nRepository/context workspace:\n")
	b.WriteString(strings.TrimSpace(task.ProjectPath))
	b.WriteString("\n\nNon-negotiable role boundary:\n")
	b.WriteString("- Do not implement application features, redesign product behaviour, or make source-code changes. ChatGPT owns coding.\n")
	b.WriteString("- You may inspect repositories and use shell/systemd/Docker/Kubernetes/Helm/Git/GitHub operational commands when they are necessary to achieve the stated objective.\n")
	b.WriteString("- Runtime/cluster changes explicitly requested by the objective (for example deploy, restart, install, apply, recover, verify) are operational work and may be carried out.\n")
	b.WriteString("- Never push or merge source changes. If a source or infrastructure-as-code edit is required, stop with WORKER_UNAVAILABLE: source change required; return to ChatGPT.\n")
	b.WriteString("- Never print, copy, or expose secret values.\n")
	b.WriteString("- Do not stop merely to report progress or ask whether to continue. Diagnose ordinary failures, retry safe alternatives, and keep working until the outcome is verified.\n")
	b.WriteString("- Ask the human only for a genuinely irreversible/destructive/production permission or product decision that is not already authorised by the objective. Use exactly ATTENTION_REQUIRED: followed by one concise question.\n")
	b.WriteString("- If login, quota, local tool policy, missing executable, or another worker-local limitation prevents you acting, use exactly WORKER_UNAVAILABLE: followed by one concise reason; do not ask the human to babysit your setup.\n")
	b.WriteString("- When and only when the operational objective is actually complete and verified, finish with a final line exactly WORKBENCH_OPERATION_COMPLETE: verified. A progress report without this marker is not completion and Workbench will automatically invoke you again.\n")

	if strings.TrimSpace(task.HumanAnswer) != "" {
		b.WriteString("\nHuman answer to the previous genuine attention request:\n")
		b.WriteString(strings.TrimSpace(task.HumanAnswer))
		b.WriteString("\nContinue from that decision; do not ask the same question again.\n")
	}
	if pass > 1 {
		b.WriteString(fmt.Sprintf("\nWorkbench supervisor continuation pass %d of %d:\n", pass, maxOperationContinuationPasses))
		b.WriteString("Your previous invocation ended without the verified-completion marker or became unresponsive. Reinspect the current host/cluster/runtime state, preserve work already done, and continue the same objective. Do not return another progress-only answer.\n")
		if previous = strings.TrimSpace(previous); previous != "" {
			b.WriteString("Previous non-secret report (context only; verify actual state yourself):\n")
			b.WriteString(previous)
			b.WriteString("\n")
		}
	}
	return b.String()
}

func runOpenClawOperationInvocation(ctx context.Context, p Provider, task Task, prefs Preferences, prompt string) (RunResult, bool, error) {
	invokeCtx, cancel := context.WithTimeout(ctx, operationInvocationTimeout)
	defer cancel()

	var name string
	var args []string
	remote := strings.TrimSpace(prefs.OpenClawSSHHost) != ""
	if remote {
		name = "ssh"
		args = []string{"-o", "BatchMode=yes", "-o", "ConnectTimeout=10", "-o", "StrictHostKeyChecking=accept-new", strings.TrimSpace(prefs.OpenClawSSHHost), "openclaw", "agent", "--message", prompt, "--headless"}
	} else if strings.TrimSpace(p.Command) != "" {
		name = p.Command
		args = []string{"agent", "--message", prompt, "--headless"}
	} else {
		return RunResult{Retryable: true}, false, errors.New("OpenClaw operations adapter is not configured")
	}

	cmd := exec.CommandContext(invokeCtx, name, args...)
	if !remote {
		cmd.Dir = task.ProjectPath
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
		return boundRunResultForPersistence(res), false, errors.New("OpenClaw operations response exceeded Workbench's bounded output limit")
	}
	if strings.TrimSpace(res.WorkerUnavailable) != "" {
		res.Retryable = true
		return boundRunResultForPersistence(res), false, fmt.Errorf("OpenClaw is unavailable for this operational task: %s", res.WorkerUnavailable)
	}
	if strings.TrimSpace(res.Attention) != "" && isWorkerSetupAttention(res.Attention) {
		q := res.Attention
		res.Attention = ""
		res.Retryable = true
		return boundRunResultForPersistence(res), false, fmt.Errorf("OpenClaw cannot act autonomously under its current local tool permissions: %s", q)
	}
	if runErr != nil {
		if errors.Is(invokeCtx.Err(), context.DeadlineExceeded) && ctx.Err() == nil {
			res.Retryable = true
			if strings.TrimSpace(res.Output) == "" {
				res.Output = "OpenClaw invocation became unresponsive before confirming completion."
			}
			return boundRunResultForPersistence(res), false, errors.New("OpenClaw operations invocation timed out")
		}
		low := strings.ToLower(out + " " + runErr.Error())
		res.Authentication = strings.Contains(low, "login") || strings.Contains(low, "sign in") || strings.Contains(low, "authenticate") || strings.Contains(low, "unauthorized") || strings.Contains(low, "credential") || strings.Contains(low, "publickey") || strings.Contains(low, "permission denied")
		res.Retryable = res.Authentication || strings.Contains(low, "not found") || strings.Contains(low, "unknown option") || strings.Contains(low, "rate limit") || strings.Contains(low, "quota") || strings.Contains(low, "timeout")
		if strings.TrimSpace(res.Output) == "" {
			res.Output = runErr.Error()
		}
		return boundRunResultForPersistence(res), false, fmt.Errorf("OpenClaw operations invocation exited with error: %w", runErr)
	}
	return boundRunResultForPersistence(res), complete, nil
}

func operationInvocationCanBeReengaged(res RunResult, err error) bool {
	if err == nil || res.Authentication || strings.TrimSpace(res.Attention) != "" || strings.TrimSpace(res.WorkerUnavailable) != "" {
		return false
	}
	low := strings.ToLower(err.Error() + " " + res.Output)
	return strings.Contains(low, "timed out") || strings.Contains(low, "timeout") || strings.Contains(low, "unresponsive")
}

func stripOperationCompletionMarker(out string) (string, bool) {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	kept := make([]string, 0, len(lines))
	complete := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToUpper(trimmed), operationCompletePrefix) {
			value := strings.TrimSpace(trimmed[len(operationCompletePrefix):])
			if strings.EqualFold(value, "verified") || strings.EqualFold(value, "yes") || strings.EqualFold(value, "complete") {
				complete = true
			}
			continue
		}
		kept = append(kept, line)
	}
	return strings.TrimSpace(strings.Join(kept, "\n")), complete
}

func operationContinuationReport(out string) string {
	out = strings.TrimSpace(out)
	if out == "" || LooksSecret(out) {
		return ""
	}
	if len(out) > maxOperationContinuationReport {
		out = out[len(out)-maxOperationContinuationReport:]
	}
	return strings.TrimSpace(out)
}
