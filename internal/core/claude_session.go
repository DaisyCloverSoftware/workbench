package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type claudePrintResponse struct {
	Result    string `json:"result"`
	SessionID string `json:"session_id"`
}

func runClaudeProvider(ctx context.Context, p Provider, task Task, project, prompt string) (RunResult, error) {
	session, hasSession, _ := ProviderSessionFor(task.ID, p.ID)
	if hasSession && !validClaudeSessionID(session.SessionID) {
		_ = DeleteProviderSession(task.ID, p.ID)
		hasSession = false
	}

	res, runErr, resumeInvalid := runClaudeInvocation(ctx, p, task, project, prompt, session.SessionID, hasSession)
	if hasSession && resumeInvalid && ctx.Err() == nil {
		_ = DeleteProviderSession(task.ID, p.ID)
		return runClaudeFresh(ctx, p, task, project, prompt)
	}
	return res, runErr
}

func runClaudeFresh(ctx context.Context, p Provider, task Task, project, prompt string) (RunResult, error) {
	res, runErr, _ := runClaudeInvocation(ctx, p, task, project, prompt, "", false)
	return res, runErr
}

func runClaudeInvocation(ctx context.Context, p Provider, task Task, project, prompt, sessionID string, resume bool) (RunResult, error, bool) {
	args := claudeInvocationArgs(prompt, sessionID, resume)
	cmd := exec.CommandContext(ctx, p.Command, args...)
	cmd.Dir = project
	configureChildProcess(cmd, false)
	stdout := newBoundedWorkerCapture(maxWorkerStreamCaptureBytes)
	stderr := newBoundedWorkerCapture(maxWorkerStreamCaptureBytes)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	runErr := cmd.Run()

	stdoutText := strings.TrimSpace(stdout.String())
	stderrText := strings.TrimSpace(stderr.String())
	combined := stdoutText
	if stderrText != "" {
		if combined != "" {
			combined += "\n\n"
		}
		combined += stderrText
	}
	if stdout.Truncated() || stderr.Truncated() {
		res := classifyRunOutput(combined)
		res.Retryable = true
		return boundRunResultForPersistence(res), errors.New("Claude Code response exceeded Workbench's bounded output limit"), false
	}

	parsed, parsedOK := parseClaudePrintResponse(stdoutText)
	if parsedOK && ctx.Err() == nil && validClaudeSessionID(parsed.SessionID) {
		_, _ = SaveProviderSession(task.ID, p.ID, parsed.SessionID)
	}
	out := combined
	if parsedOK && strings.TrimSpace(parsed.Result) != "" {
		out = strings.TrimSpace(parsed.Result)
		if stderrText != "" && runErr != nil {
			out += "\n\n" + stderrText
		}
	} else {
		out = normalizeProviderOutput(p.ID, combined)
	}
	res := classifyRunOutput(out)

	if resume && runErr != nil && claudeResumeUnavailable(combined) {
		res.Retryable = true
		return boundRunResultForPersistence(res), fmt.Errorf("Claude Code stored session is unavailable: %w", runErr), true
	}
	if strings.TrimSpace(res.WorkerUnavailable) != "" {
		res.Retryable = true
		return boundRunResultForPersistence(res), fmt.Errorf("%s is unavailable for this task: %s", p.Name, res.WorkerUnavailable), false
	}
	if strings.TrimSpace(res.Attention) != "" && isWorkerSetupAttention(res.Attention) {
		q := res.Attention
		res.Attention = ""
		res.Retryable = true
		return boundRunResultForPersistence(res), fmt.Errorf("%s cannot act autonomously under its current local tool permissions: %s", p.Name, q), false
	}
	if runErr != nil {
		low := strings.ToLower(combined + " " + runErr.Error())
		res.Authentication = strings.Contains(low, "login") || strings.Contains(low, "sign in") || strings.Contains(low, "authenticate") || strings.Contains(low, "unauthorized") || strings.Contains(low, "credential")
		res.Retryable = res.Authentication || strings.Contains(low, "not found") || strings.Contains(low, "unknown option") || strings.Contains(low, "sandbox") || strings.Contains(low, "rate limit") || strings.Contains(low, "quota")
		if strings.TrimSpace(res.Output) == "" {
			res.Output = runErr.Error()
		}
		return boundRunResultForPersistence(res), fmt.Errorf("%s exited with error: %w", p.Name, runErr), false
	}
	res.Output = persistWorkerMemories(task, res.Output)
	return boundRunResultForPersistence(res), nil, false
}

func claudeInvocationArgs(prompt, sessionID string, resume bool) []string {
	args := []string{"-p", prompt, "--output-format", "json", "--permission-mode", "acceptEdits", "--allowedTools"}
	args = append(args, claudeAllowedTools()...)
	if resume {
		args = append(args, "--resume", sessionID)
	}
	return args
}

func parseClaudePrintResponse(out string) (claudePrintResponse, bool) {
	out = strings.TrimSpace(out)
	if out == "" {
		return claudePrintResponse{}, false
	}
	var response claudePrintResponse
	if err := json.Unmarshal([]byte(out), &response); err != nil {
		return claudePrintResponse{}, false
	}
	response.Result = strings.TrimSpace(response.Result)
	response.SessionID = strings.TrimSpace(response.SessionID)
	return response, response.SessionID != "" || response.Result != ""
}

func validClaudeSessionID(id string) bool {
	id = strings.TrimSpace(id)
	if len(id) != 36 {
		return false
	}
	for i, r := range id {
		switch i {
		case 8, 13, 18, 23:
			if r != '-' {
				return false
			}
		default:
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
				return false
			}
		}
	}
	return true
}

func claudeResumeUnavailable(output string) bool {
	low := strings.ToLower(strings.TrimSpace(output))
	if !strings.Contains(low, "session") && !strings.Contains(low, "conversation") {
		return false
	}
	return strings.Contains(low, "not found") ||
		strings.Contains(low, "does not exist") ||
		strings.Contains(low, "no conversation") ||
		strings.Contains(low, "invalid session") ||
		strings.Contains(low, "unknown session") ||
		strings.Contains(low, "cannot resume") ||
		strings.Contains(low, "can't resume")
}
