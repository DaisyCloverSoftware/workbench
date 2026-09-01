package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	operationCompletePrefix         = "WORKBENCH_OPERATION_COMPLETE:"
	maxOperationContinuationPasses  = 6
	operationInvocationTimeout       = 10 * time.Minute
	maxOperationContinuationReport  = 4000
	defaultOpenClawOperationsAgent  = "main"
	openClawOperationArchiveTimeout = 15 * time.Second
)

// RunOpenClawOperationSupervised runs only an explicitly owner-authorized
// OpenClaw machine operation. A clean OpenClaw process exit is not considered
// task completion unless OpenClaw has explicitly verified the requested
// operational outcome. Progress-only exits and bounded unresponsive invocations
// are re-engaged automatically against the same real host/cluster state instead
// of making the human type "continue".
//
// Every authorized durable Workbench task owns one explicit OpenClaw
// conversation. All continuation passes and model failover attempts for that
// task reuse the same session, while a different task receives a different
// session. This preserves useful per-job context without inheriting stale
// bindings from OpenClaw's long-lived interactive main conversation.
func RunOpenClawOperationSupervised(ctx context.Context, p Provider, task Task, prefs Preferences) (RunResult, error) {
	if !task.OpenClawOwnerAuthorized {
		return RunResult{}, errors.New("OpenClaw authorization denied: durable task state does not contain explicit owner authorization naming OpenClaw")
	}
	if p.ID != "openclaw" {
		return RunResult{}, errors.New("explicitly owner-authorized OpenClaw operations lane requires the OpenClaw provider")
	}
	sessionID := openClawOperationSessionID(task)
	previous := ""
	for pass := 1; pass <= maxOperationContinuationPasses; pass++ {
		if err := ctx.Err(); err != nil {
			return RunResult{}, err
		}
		prompt := BuildOpenClawOperationPrompt(task, pass, previous)
		res, complete, runErr := runOpenClawOperationInvocationWithFallback(ctx, p, task, prefs, prompt, sessionID)
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
			archiveOpenClawOperationSessionBestEffort(p, task, prefs, sessionID)
			return boundRunResultForPersistence(res), nil
		}
		previous = operationContinuationReport(res.Output)
	}
	res := RunResult{Output: previous, Retryable: true}
	return boundRunResultForPersistence(res), fmt.Errorf("OpenClaw stopped %d times without verifying the explicitly owner-authorized operational objective; Workbench exhausted its automatic continuation budget instead of asking the human to keep nudging it", maxOperationContinuationPasses)
}

func BuildOpenClawOperationPrompt(task Task, pass int, previous string) string {
	var b strings.Builder
	b.WriteString("You are OpenClaw acting only on a machine-side operation the owner explicitly assigned to OpenClaw by name. ChatGPT owns the software-development loop: reasoning, source code, Git/GitHub changes, pull requests, CI, GitHub Actions, releases, and subsequent engineering decisions. Your authority is limited to the explicitly owner-authorized machine-operation objective below. Do not infer or expand authority from tool availability, task difficulty, previous OpenClaw use, or limitations of Workbench's direct controls.\n\n")
	b.WriteString("Operational objective:\n")
	b.WriteString(OperationsTaskIntent(task))
	b.WriteString("\n\nRepository/context workspace:\n")
	b.WriteString(strings.TrimSpace(task.ProjectPath))
	b.WriteString("\n\nNon-negotiable role boundary:\n")
	b.WriteString("- Do not implement application features, redesign product behaviour, make source-code or infrastructure-as-code changes, create commits/branches/PRs, push or merge, trigger/rerun CI, or operate GitHub Actions. ChatGPT owns all of that.\n")
	b.WriteString("- You may inspect repository state read-only when needed to identify the deployed revision, but do not mutate Git or GitHub state.\n")
	b.WriteString("- You may use shell/systemd/Docker/Kubernetes/Helm and equivalent host/runtime commands only when they are necessary to achieve this explicitly owner-authorized objective and remain within the existing authority boundary.\n")
	b.WriteString("- Runtime/cluster changes explicitly requested by the objective (for example deploy an already-built artifact, restart, install, apply, recover, or verify) are operational work and may be carried out.\n")
	b.WriteString("- If completing the objective requires any code/IaC/GitHub/CI change, stop with WORKER_UNAVAILABLE: ChatGPT-owned development change required; return to ChatGPT.\n")
	b.WriteString("- Never print, copy, or expose secret values.\n")
	b.WriteString("- Do not stop merely to report progress or ask whether to continue. Diagnose ordinary operational failures, retry safe alternatives, and keep working until the authorized outcome is verified.\n")
	b.WriteString("- Ask the human only for a genuinely irreversible/destructive/production permission or product decision that is not already authorised by the objective. Use exactly ATTENTION_REQUIRED: followed by one concise question.\n")
	b.WriteString("- If login, quota, local tool policy, missing executable, or another worker-local limitation prevents you acting, use exactly WORKER_UNAVAILABLE: followed by one concise reason; do not ask the human to babysit your setup.\n")
	b.WriteString("- When and only when the operational objective is actually complete and verified, finish with a final line exactly WORKBENCH_OPERATION_COMPLETE: verified. A progress report without this marker is not completion and Workbench will automatically invoke you again within this same already-authorized task.\n")

	if strings.TrimSpace(task.HumanAnswer) != "" {
		b.WriteString("\nHuman answer to the previous genuine attention request:\n")
		b.WriteString(strings.TrimSpace(task.HumanAnswer))
		b.WriteString("\nContinue from that decision; do not ask the same question again.\n")
	}
	if pass > 1 {
		b.WriteString(fmt.Sprintf("\nWorkbench supervisor continuation pass %d of %d:\n", pass, maxOperationContinuationPasses))
		b.WriteString("Your previous invocation ended without the verified-completion marker or became unresponsive. Reinspect the current host/cluster/runtime state, preserve operational work already done, and continue the same explicitly owner-authorized objective in this same Workbench job conversation. Do not return another progress-only answer.\n")
		if previous = strings.TrimSpace(previous); previous != "" {
			b.WriteString("Previous non-secret report (context only; verify actual state yourself):\n")
			b.WriteString(previous)
			b.WriteString("\n")
		}
	}
	return b.String()
}

// openClawOperationSessionID is deterministic for one durable Workbench task
// and deliberately opaque. Automatic retries, attention resumes and supervisor
// continuation passes therefore reconnect to the same OpenClaw conversation;
// a different Workbench task gets a different conversation.
func openClawOperationSessionID(task Task) string {
	identity := strings.TrimSpace(task.ID)
	if identity == "" {
		identity = strings.TrimSpace(task.ProjectPath) + "\x00" + strings.TrimSpace(OperationsTaskIntent(task))
	}
	sum := sha256.Sum256([]byte(identity))
	return "workbench-op-" + hex.EncodeToString(sum[:12])
}

func openClawOperationSessionKeyForID(sessionID string) string {
	return "agent:" + defaultOpenClawOperationsAgent + ":explicit:" + strings.TrimSpace(sessionID)
}

func openClawOperationSessionKey(task Task) string {
	return openClawOperationSessionKeyForID(openClawOperationSessionID(task))
}

func openClawOperationAgentArgs(task Task, prompt string) []string {
	return openClawOperationAgentArgsWithSession(prompt, "", openClawOperationSessionID(task))
}

func openClawOperationAgentArgsForModel(task Task, prompt, model string) []string {
	return openClawOperationAgentArgsWithSession(prompt, model, openClawOperationSessionID(task))
}

func openClawOperationAgentArgsWithSession(prompt, model, sessionID string) []string {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil
	}
	args := []string{"agent", "--agent", defaultOpenClawOperationsAgent, "--session-id", sessionID}
	if model = strings.TrimSpace(model); model != "" {
		args = append(args, "--model", model)
	}
	return append(args, "--message", prompt)
}

func runOpenClawOperationInvocation(ctx context.Context, p Provider, task Task, prefs Preferences, prompt, sessionID string) (RunResult, bool, error) {
	if !task.OpenClawOwnerAuthorized {
		return RunResult{}, false, errors.New("OpenClaw authorization denied before process invocation")
	}
	invokeCtx, cancel := context.WithTimeout(ctx, operationInvocationTimeout)
	defer cancel()
	if strings.TrimSpace(sessionID) == "" || sessionID != openClawOperationSessionID(task) {
		return RunResult{}, false, errors.New("OpenClaw operations session identity mismatch")
	}

	var name string
	var args []string
	remote := strings.TrimSpace(prefs.OpenClawSSHHost) != ""
	if remote {
		name = "ssh"
		args = []string{"-o", "BatchMode=yes", "-o", "ConnectTimeout=10", "-o", "StrictHostKeyChecking=accept-new", strings.TrimSpace(prefs.OpenClawSSHHost), "openclaw"}
		args = append(args, openClawOperationAgentArgsWithSession(prompt, "", sessionID)...)
	} else if strings.TrimSpace(p.Command) != "" {
		name = p.Command
		args = openClawOperationAgentArgsWithSession(prompt, "", sessionID)
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
		return boundRunResultForPersistence(res), false, errors.New("OpenClaw operations response exceeded Workbench's bounded output limit")
	}
	if strings.TrimSpace(res.WorkerUnavailable) != "" {
		res.Retryable = true
		return boundRunResultForPersistence(res), false, fmt.Errorf("OpenClaw is unavailable for this explicitly owner-authorized operational task: %s", res.WorkerUnavailable)
	}
	if strings.TrimSpace(res.Attention) != "" && isWorkerSetupAttention(res.Attention) {
		q := res.Attention
		res.Attention = ""
		res.Retryable = true
		return boundRunResultForPersistence(res), false, fmt.Errorf("OpenClaw cannot act under its current local tool permissions: %s", q)
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

func archiveOpenClawOperationSessionBestEffort(p Provider, task Task, prefs Preferences, sessionID string) {
	if strings.TrimSpace(sessionID) == "" || sessionID != openClawOperationSessionID(task) {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), openClawOperationArchiveTimeout)
	defer cancel()
	key := openClawOperationSessionKeyForID(sessionID)
	remote := strings.TrimSpace(prefs.OpenClawSSHHost) != ""
	var cmd *exec.Cmd
	if remote {
		cmd = exec.CommandContext(ctx, "ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=10", "-o", "StrictHostKeyChecking=accept-new", strings.TrimSpace(prefs.OpenClawSSHHost), "openclaw", "sessions", "archive", key)
	} else if strings.TrimSpace(p.Command) != "" {
		cmd = exec.CommandContext(ctx, p.Command, "sessions", "archive", key)
		cmd.Env = environmentForUserExecutable(p.Command)
	} else {
		return
	}
	configureChildProcess(cmd, false)
	_ = cmd.Run()
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
